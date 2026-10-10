# Quantity Discount Calculator Lambda

This standalone Go Lambda calculates a quantity discount for an existing checkout item. Quantity `5` or greater receives a 10% discount. The gateway should call it before inventory deduction during a later integration step.

## Contract

Request fields are `productId`, `quantity`, and `unitPrice`. Prices are converted to integer cents and rounded half-up before calculation. Invalid input returns status `400` in the response payload; it does not return a Go error.

## Build and package

This project targets ARM64. Choose **arm64** when creating the Lambda function in AWS and use runtime **Amazon Linux 2023 (provided.al2023)** with handler **bootstrap**.

```bash
go test ./...
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap .
zip function.zip bootstrap
```

The same commands are available through the Makefile:

```bash
make test
make package
```

To build for x86_64 instead, use `make package ARCH=amd64` and select **x86_64** in the AWS console. The architecture used to build and the architecture configured for the function must match.

## AWS console settings

- Runtime: `Amazon Linux 2023 (provided.al2023)`
- Handler: `bootstrap`
- Architecture: `arm64`

Upload `function.zip` and use the events in `test-events.json` in the Test tab.
