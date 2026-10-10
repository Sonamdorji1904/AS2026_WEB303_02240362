package main

import (
	"context"
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
)

const (
	maxQuantity       int64 = 10000
	discountThreshold int64 = 5
	discountRate            = 0.10
)

type Request struct {
	ProductID string  `json:"productId"`
	Quantity  int64   `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Response struct {
	StatusCode int         `json:"statusCode"`
	Body       interface{} `json:"body"`
}

type SuccessBody struct {
	ProductID    string  `json:"productId"`
	Quantity     int64   `json:"quantity"`
	UnitPrice    Money   `json:"unitPrice"`
	Subtotal     Money   `json:"subtotal"`
	DiscountRate float64 `json:"discountRate"`
	Discount     Money   `json:"discount"`
	Total        Money   `json:"total"`
}

type ErrorBody struct {
	Error string `json:"error"`
}

// Money is backed by a float for the public JSON contract, but is calculated
// from integer cents before it is converted to this type.
type Money float64

func (money Money) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(money), 'f', 2, 64)), nil
}

func handler(_ context.Context, request Request) (Response, error) {
	unitPriceCents, validationError := validateRequest(request)
	if validationError != nil {
		return Response{
			StatusCode: 400,
			Body:       ErrorBody{Error: validationError.Error()},
		}, nil
	}

	subtotalCents := unitPriceCents * request.Quantity
	discountCents := int64(0)
	if request.Quantity >= discountThreshold {
		// Round the 10% discount half-up to the nearest cent.
		discountCents = (subtotalCents + 5) / 10
	}

	return Response{
		StatusCode: 200,
		Body: SuccessBody{
			ProductID:    request.ProductID,
			Quantity:     request.Quantity,
			UnitPrice:    moneyFromCents(unitPriceCents),
			Subtotal:     moneyFromCents(subtotalCents),
			DiscountRate: discountRateFor(request.Quantity),
			Discount:     moneyFromCents(discountCents),
			Total:        moneyFromCents(subtotalCents - discountCents),
		},
	}, nil
}

func validateRequest(request Request) (int64, error) {
	if strings.TrimSpace(request.ProductID) == "" {
		return 0, errors.New("productId is required")
	}
	if request.Quantity < 1 || request.Quantity > maxQuantity {
		return 0, errors.New("quantity must be an integer between 1 and 10000")
	}
	if request.UnitPrice <= 0 || math.IsNaN(request.UnitPrice) || math.IsInf(request.UnitPrice, 0) {
		return 0, errors.New("unitPrice must be a finite number greater than zero")
	}

	unitPriceCents, err := decimalToCents(request.UnitPrice)
	if err != nil {
		return 0, err
	}
	if unitPriceCents <= 0 {
		return 0, errors.New("unitPrice must round to at least one cent")
	}
	maxInt64 := int64(^uint64(0) >> 1)
	if unitPriceCents > maxInt64/request.Quantity {
		return 0, errors.New("order total is too large")
	}
	return unitPriceCents, nil
}

func decimalToCents(value float64) (int64, error) {
	decimal := strconv.FormatFloat(value, 'f', -1, 64)
	rational, ok := new(big.Rat).SetString(decimal)
	if !ok {
		return 0, errors.New("unitPrice must be a valid decimal")
	}

	rational.Mul(rational, big.NewRat(100, 1))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(rational.Num(), rational.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(rational.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}

	maxInt64 := new(big.Int).SetUint64(^uint64(0) >> 1)
	if quotient.Sign() <= 0 || quotient.Cmp(maxInt64) > 0 {
		return 0, errors.New("unitPrice is outside the supported range")
	}
	return quotient.Int64(), nil
}

func discountRateFor(quantity int64) float64 {
	if quantity >= discountThreshold {
		return discountRate
	}
	return 0
}

func moneyFromCents(cents int64) Money {
	return Money(float64(cents) / 100)
}

func main() {
	lambda.Start(handler)
}
