# WEB303 Lab 4 Report
## Developing a Comprehensive Test Suite for a Microservice Application

**Application:** E-Commerce Microservices System  
**Module:** WEB303  
**Practical:** Lab 4  
**Date:** 27 August 2026

## 1. Aim

The aim of this practical was to extend an existing e-commerce microservices application with a user interface and develop a comprehensive test suite covering individual services, communication between services, and complete user-facing workflows.

## 2. Objectives

- Extend the application with a simple web-based user interface.
- Allow users to search for products by identifier and view product details.
- Display suitable messages when a record does not exist or a service is unavailable.
- Exercise the Customer, Inventory, Payment, and Product services through the gateway.
- Develop independent unit tests for each core service.
- Develop integration tests for successful responses, service errors, unavailable dependencies, and timeouts.
- Develop end-to-end tests for the main UI and API workflows.
- Test retry and circuit-breaker behavior from the previous practical.
- Record the expected and actual results of the tests.

## 3. System Overview

The application contains five service programs and a protocol definition:

| Component | Responsibility | Interface / Port |
|---|---|---|
| Product Service | Retrieves product information | gRPC, `50051` |
| Customer Service | Retrieves customer information | gRPC, `50052` |
| Inventory Service | Checks and deducts stock | gRPC, `50053` |
| Payment Service | Processes payments | gRPC, `50054` |
| Order Service | Demonstrates order integration, retry, and circuit breaker logic | Command-line workflow |
| Gateway | Serves the UI and translates HTTP requests to gRPC | HTTP, `8080` |

The gRPC contracts are defined in `proto/product.proto`. The gateway communicates with the services through generated gRPC clients. It does not read any service's internal map or data store directly.

### Order-service adaptation

The current `order-service` is a command-line integration demonstration and does not expose an `OrderService` RPC in the protocol definition. Therefore, the gateway provides the user-facing order workflow by calling the existing Product, Customer, Inventory, and Payment interfaces in sequence:

1. Retrieve the product.
2. Validate the customer.
3. Deduct inventory.
4. Process payment.
5. Return the transaction and remaining stock to the UI.

This is the documented adaptation of the general lab requirements to the interfaces available in this application.

## 4. Part A: User Interface

The UI is implemented in `gateway/static/index.html` and served by `gateway/main.go`.

The landing page provides:

- A visual product catalog.
- A horizontal product search bar.
- Product details and prices.
- Product-specific SVG icons.
- An `Order this` action for each product.
- Customer lookup by customer ID.
- Inventory checks by product ID and quantity.
- An order form using product ID, customer ID, and quantity.
- Success, validation, missing-record, timeout, and unavailable-service messages.

The product cards use SVG assets embedded by the gateway. The UI calls endpoints such as `/api/products`, `/api/customers`, `/api/inventory`, and `/api/orders`; the gateway then calls the corresponding gRPC service interfaces.

## 5. Part B: Unit Testing

Unit tests are located beside the service implementations:

- `product-service/main_test.go`
- `customer-service/main_test.go`
- `inventory-service/main_test.go`
- `payment-service/main_test.go`
- `order-service/main_test.go`

The unit tests run independently and do not require service processes or network ports.

### Unit test coverage

- Existing product retrieval for `P001` and `P004`.
- Different customer retrieval for `C001` and `C002`.
- Missing product and customer identifiers.
- Empty product and customer identifiers.
- Inventory checks and stock deduction.
- Missing stock records.
- Zero or invalid quantities.
- Insufficient inventory.
- Successful and failed payment status results.
- Invalid payment requests.
- Non-retryable product errors.
- Circuit breaker opening after repeated failures.

Input validation was added to the Product, Customer, Inventory, and Payment services. Invalid requests return gRPC `InvalidArgument`, missing records return `NotFound`, and insufficient stock returns `FailedPrecondition` where appropriate.

## 6. Part C: Integration Testing

Gateway integration tests are located in `gateway/main_test.go`. Test doubles implement the generated gRPC client interfaces so the tests are repeatable and independent of running services.

The integration tests cover:

- Product dependency available.
- Product dependency returning `NotFound`.
- Product dependency returning `Unavailable`.
- Product dependency exceeding the configured context timeout.
- Successful order orchestration across product, customer, inventory, and payment clients.
- Product dependency timeout during an order request.
- Correct conversion of gRPC errors into HTTP status codes and user-facing messages.

The gateway uses a six-second request context. Timeout and unavailable errors are returned to the UI as HTTP `503 Service Unavailable` responses.

## 7. Part D: End-to-End Testing

End-to-end HTTP tests are located in `gateway/e2e_test.go`. They run the gateway handlers through an `httptest` server and verify the user-facing page and API responses.

The tests cover:

1. Landing page renders the product-first UI.
2. Existing product identifier `P001` returns the MacBook product.
3. A different or missing identifier such as `P999` returns a not-found response.
4. An empty identifier returns a validation response.
5. A successful order workflow returns a payment transaction and inventory result through the gateway integration tests.
6. Timeout behavior is verified through a deliberately slow service double.
7. Retry behavior is tested in `order-service/main_test.go`.
8. Circuit-breaker behavior is tested by generating repeated failures and verifying that the circuit opens.

For a live manual demonstration, the services can be started in separate terminals:

```bash
go run ./product-service
go run ./customer-service
go run ./inventory-service
go run ./payment-service
go run ./gateway
```

The UI is then available at `http://localhost:8080`.

To demonstrate service unavailability manually, stop one of the gRPC services and submit the related request through the UI. The gateway displays an appropriate failure message.

## 8. Part E: Test Results

The complete automated suite was executed using:

```bash
go test ./...
```

### Test Log

| # | Test Case Description | Input / Condition | Expected Result | Actual Result | Status |
|---:|---|---|---|---|---|
| 1 | Product unit test: existing record | Request `P001` | Product details are returned | MacBook product details returned | PASS |
| 2 | Product unit test: different record | Request `P004` | A different product is returned correctly | 4K Monitor details returned | PASS |
| 3 | Customer unit test: existing records | Requests `C001` and `C002` | Correct customer data is returned for each ID | Both customer records matched expected data | PASS |
| 4 | Product/customer unit test: missing record | Requests `P999` and `C999` | `NotFound` error is returned | `NotFound` returned | PASS |
| 5 | Product/customer unit test: invalid identifier | Empty product or customer ID | `InvalidArgument` error is returned | `InvalidArgument` returned | PASS |
| 6 | Inventory unit test: stock check and deduction | Check and deduct 3 units of `P001` | Availability and remaining stock are correct | Stock changed from 10 to 7 | PASS |
| 7 | Inventory unit test: error handling | Missing ID, zero quantity, and insufficient stock | Appropriate errors are returned | `NotFound`, `InvalidArgument`, and `FailedPrecondition` returned | PASS |
| 8 | Payment unit test: success and business failure | Normal payment and amount `500001` | Payment returns `SUCCESS` or `FAILED` appropriately | Expected statuses returned | PASS |
| 9 | Gateway integration: dependency available | Mock ProductService returns a product | HTTP `200` with product details | Product response returned | PASS |
| 10 | Gateway integration: dependency error | Mock ProductService returns `NotFound` | HTTP `404` with a user-facing message | HTTP `404` and message returned | PASS |
| 11 | Gateway integration: dependency unavailable | Mock ProductService returns `Unavailable` | HTTP `503` with an unavailable message | HTTP `503` and message returned | PASS |
| 12 | Gateway integration: dependency timeout | Mock ProductService exceeds the request context | HTTP `503` with a timeout message | HTTP `503` and message returned | PASS |
| 13 | Gateway integration: order workflow | Product, customer, inventory, and payment mocks succeed | Order returns transaction and stock result | HTTP `200`, transaction, and stock returned | PASS |
| 14 | End-to-end: landing page | `GET /` | Product-first UI is displayed | Catalog page returned successfully | PASS |
| 15 | End-to-end: existing record | `GET /api/products?id=P001` | Correct product information is displayed | MacBook product returned | PASS |
| 16 | End-to-end: non-existing record | `GET /api/products?id=P999` | Appropriate not-found message is displayed | HTTP `404` and message returned | PASS |
| 17 | End-to-end: invalid input | `GET /api/products?id=` | Appropriate validation message is displayed | HTTP `400` and message returned | PASS |
| 18 | Retry behavior | Missing product passed to retry function | Non-retryable error ends retry processing correctly | `NotFound` returned without retrying | PASS |
| 19 | Circuit breaker behavior | Three forced dependency failures | Circuit opens and blocks further calls | Circuit state became `open` | PASS |

All nineteen recorded test cases passed.

The final command completed successfully for all packages:

```text
ok  ecommerce-microservices/customer-service
ok  ecommerce-microservices/gateway
ok  ecommerce-microservices/inventory-service
ok  ecommerce-microservices/order-service
ok  ecommerce-microservices/payment-service
ok  ecommerce-microservices/product-service
?   ecommerce-microservices/proto  [no test files]
```

A detailed test matrix containing test descriptions, inputs, expected results, actual results, and statuses is available in `TEST_RESULTS.md`.

## 9. Discussion

The test suite demonstrates three levels of testing:

- **Unit testing** isolates service logic and validates records, errors, validation, and business rules.
- **Integration testing** validates the gateway's communication behavior using controlled service doubles.
- **End-to-end testing** validates the UI-facing HTTP behavior, including the landing page, product search, invalid input, and missing records.

The gateway also provides a useful boundary for failure handling. gRPC errors are translated into HTTP responses that can be understood by a browser user. This makes service failures visible without exposing internal service implementation details.

The use of test doubles makes integration and failure tests repeatable. A test does not depend on random service state, running ports, or manually stopping a process.

## 10. Limitations and Future Improvements

- The Order Service is currently a command-line program rather than a networked gRPC service. Adding an `OrderService` RPC would allow the gateway to call it directly.
- The automated end-to-end tests use an HTTP test server and service doubles. A browser automation test against all live services could be added for a stricter full-system demonstration.
- The Product Service contains an optional three-second chaos delay. A configurable delay or environment variable would make live timeout demonstrations easier.
- The current services use in-memory data, so persistence and concurrent deployment are outside the scope of this practical.

## 11. Conclusion

The e-commerce microservices application was extended with a functional UI and tested at unit, integration, and end-to-end levels. The tests cover normal operations, different identifiers, missing records, invalid input, unavailable dependencies, timeouts, retry behavior, and circuit-breaker behavior. All automated tests pass, and the recorded results provide evidence that the application handles both successful and unsuccessful execution paths correctly.
