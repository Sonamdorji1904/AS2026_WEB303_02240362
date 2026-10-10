# WEB303 Lab 4 Test Results

The test suite is repeatable and does not require the five services to be running for unit, integration, or HTTP end-to-end tests.

## Part A: UI

The UI is served by `gateway` at `http://localhost:8080`. It communicates with Product, Customer, Inventory, and Payment through gRPC clients in `gateway/main.go`. It does not access service data stores directly.

The order form is an adapted order workflow because `order-service` is currently a command-line program and `product.proto` does not define an OrderService RPC. The gateway performs product validation, customer validation, inventory deduction, and payment through the available service interfaces.

## Automated Test Log

| # | Test Case Description | Input / Condition | Expected Result | Actual Result | Status |
|---:|---|---|---|---|---|
| 1 | Product unit test: existing record | `P001` | Product details returned | MacBook details matched | PASS |
| 2 | Product unit test: different record | `P004` | Different product returned | Monitor details matched | PASS |
| 3 | Customer unit test: different records | `C001`, `C002` | Correct customer data returned | Both records matched | PASS |
| 4 | Missing record handling | `P999`, `C999` | `NotFound` returned | `NotFound` returned | PASS |
| 5 | Invalid identifier handling | Empty IDs | `InvalidArgument` returned | `InvalidArgument` returned | PASS |
| 6 | Inventory success | Check/deduct 3 units of `P001` | Correct availability and stock | Stock changed 10 to 7 | PASS |
| 7 | Inventory errors | Missing ID, zero quantity, insufficient stock | Correct errors returned | Expected errors returned | PASS |
| 8 | Payment success/failure | Normal payment, amount `500001` | `SUCCESS` / `FAILED` | Expected statuses returned | PASS |
| 9 | Dependency available | Product mock succeeds | HTTP `200` and details | Passed | PASS |
| 10 | Dependency returns error | Product mock returns `NotFound` | HTTP `404` and message | Passed | PASS |
| 11 | Dependency unavailable | Product mock returns `Unavailable` | HTTP `503` and message | Passed | PASS |
| 12 | Dependency timeout | Product mock exceeds context | HTTP `503` and timeout message | Passed | PASS |
| 13 | Order integration | All service mocks succeed | Transaction and stock result | Passed | PASS |
| 14 | UI landing page | `GET /` | Product-first UI rendered | Passed | PASS |
| 15 | UI existing record | `GET /api/products?id=P001` | Product displayed | MacBook returned | PASS |
| 16 | UI missing record | `GET /api/products?id=P999` | Not-found message | HTTP `404` returned | PASS |
| 17 | UI invalid input | Empty product ID | Validation message | HTTP `400` returned | PASS |
| 18 | Retry behavior | Missing product | Non-retryable error ends retries | Passed | PASS |
| 19 | Circuit breaker behavior | Three forced failures | Circuit opens | State became `open` | PASS |

## Commands Executed

```bash
go test ./...
```

Result: all packages passed, including unit, gateway integration, gateway end-to-end, retry, and circuit-breaker tests.

## Live UI Scenarios

For manual demonstration, start the services in separate terminals:

```bash
go run ./product-service
go run ./customer-service
go run ./inventory-service
go run ./payment-service
go run ./gateway
```

Then open `http://localhost:8080`. To demonstrate an unavailable dependency, stop one gRPC service and submit its related UI request. The gateway returns a service-unavailable response. The gateway tests use isolated doubles for this condition so the automated suite remains repeatable.
