FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/product-service ./product-service

FROM alpine:3.22
RUN adduser -D -H appuser && mkdir -p /data && chown -R appuser:appuser /data
USER appuser
WORKDIR /app
COPY --from=builder /out/product-service /app/product-service
EXPOSE 50051
ENTRYPOINT ["/app/product-service"]
