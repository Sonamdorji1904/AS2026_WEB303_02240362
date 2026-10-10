package service

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "ecommerce-microservices/proto"
)

type ResilienceConfig struct {
	CallTimeout    time.Duration
	MaxRetries     int
	BaseBackoff    time.Duration
	BreakerTrips   uint32
	BreakerTimeout time.Duration
}

func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		CallTimeout:    2 * time.Second,
		MaxRetries:     3,
		BaseBackoff:    100 * time.Millisecond,
		BreakerTrips:   3,
		BreakerTimeout: 10 * time.Second,
	}
}

func ResilienceConfigFromEnv(getenv func(string) string) ResilienceConfig {
	config := DefaultResilienceConfig()
	config.CallTimeout = durationEnv(getenv("PRODUCT_CALL_TIMEOUT"), config.CallTimeout)
	config.MaxRetries = positiveIntEnv(getenv("PRODUCT_MAX_RETRIES"), config.MaxRetries)
	config.BaseBackoff = durationEnv(getenv("PRODUCT_RETRY_BACKOFF"), config.BaseBackoff)
	config.BreakerTrips = uint32(positiveIntEnv(getenv("PRODUCT_BREAKER_FAILURES"), int(config.BreakerTrips)))
	config.BreakerTimeout = durationEnv(getenv("PRODUCT_BREAKER_TIMEOUT"), config.BreakerTimeout)
	return config
}

type ResilientProductClient struct {
	client  ProductClient
	config  ResilienceConfig
	breaker *gobreaker.CircuitBreaker
}

func NewResilientProductClient(client ProductClient, config ResilienceConfig) *ResilientProductClient {
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "product-service",
		Timeout:     config.BreakerTimeout,
		MaxRequests: 1,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= config.BreakerTrips
		},
		// Business errors are successful from the breaker perspective: they
		// must not open the circuit when Product service is healthy.
		IsSuccessful: func(err error) bool {
			return !isRetryableProductError(err)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("[CIRCUIT BREAKER] %s: %s -> %s", name, from, to)
		},
	})
	return &ResilientProductClient{client: client, config: config, breaker: breaker}
}

func (c *ResilientProductClient) GetProduct(ctx context.Context, req *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.Product, error) {
	result, err := c.breaker.Execute(func() (interface{}, error) {
		product, err := c.getWithRetry(ctx, req, opts...)
		if err != nil {
			return nil, err
		}
		return product, nil
	})
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, status.Error(codes.Unavailable, "product service circuit breaker is open")
		}
		return nil, err
	}
	return result.(*pb.Product), nil
}

func (c *ResilientProductClient) getWithRetry(ctx context.Context, req *pb.GetProductRequest, opts ...grpc.CallOption) (*pb.Product, error) {
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, status.Error(codes.DeadlineExceeded, "product lookup deadline exceeded")
		}

		callCtx, cancel := context.WithTimeout(ctx, c.config.CallTimeout)
		product, err := c.client.GetProduct(callCtx, req, opts...)
		cancel()
		if err == nil {
			return product, nil
		}
		lastErr = err
		if !isRetryableProductError(err) || attempt == c.config.MaxRetries {
			return nil, err
		}

		backoff := c.config.BaseBackoff * time.Duration(1<<attempt)
		log.Printf("[PRODUCT RETRY] attempt %d/%d failed with %s; backing off %s", attempt+1, c.config.MaxRetries+1, status.Code(err), backoff)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, status.Error(codes.DeadlineExceeded, "product lookup deadline exceeded during retry")
		case <-timer.C:
		}
	}
	return nil, lastErr
}

func isRetryableProductError(err error) bool {
	code := status.Code(err)
	return code == codes.Unavailable || code == codes.DeadlineExceeded
}

func durationEnv(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func positiveIntEnv(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
