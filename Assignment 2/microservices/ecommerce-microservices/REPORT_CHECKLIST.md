# PDF report screenshot checklist

Take the following screenshots while demonstrating the completed assignment. Keep the terminal prompt, command, response status, and relevant logs visible where possible.

| # | Screenshot | Suggested caption | Explanation |
|---:|---|---|---|
| 1 | `docker compose up --build` starting all containers | Docker Compose starts the complete microservices stack | Shows the gateway, Product service, and Order service images being built and started. |
| 2 | `docker compose ps` with all services healthy | All required containers are running and healthy | Proves the stack has exactly two services plus the gateway with healthchecks. |
| 3 | Postman `POST /products` | Product creation returns HTTP 201 | Demonstrates Product CRUD create through the REST gateway. |
| 4 | Postman `GET /products/{id}` | Product retrieval returns HTTP 200 | Demonstrates retrieval of a persisted product. |
| 5 | Postman `PUT /products/{id}` | Product update returns HTTP 200 | Demonstrates Product update. |
| 6 | Postman `DELETE /products/{id}` | Product deletion returns HTTP 204 | Demonstrates Product deletion. |
| 7 | Postman `POST /orders` | Order creation returns HTTP 201 | Shows Order service validation of the Product through gRPC before storing the order. |
| 8 | Postman `GET /orders/{id}` | Order retrieval returns HTTP 200 | Demonstrates Order retrieval and its Product dependency check. |
| 9 | Postman `PUT /orders/{id}` | Order update returns HTTP 200 | Demonstrates Order update with Product validation. |
| 10 | Postman `DELETE /orders/{id}` | Order deletion returns HTTP 204 | Demonstrates Order deletion. |
| 11 | Invalid Product request | Invalid Product data is rejected with HTTP 400 | Shows empty name, non-positive price, or negative stock is never stored. |
| 12 | Invalid Order request | Invalid Order data is rejected with HTTP 400 | Shows missing Product ID or non-positive quantity is rejected. |
| 13 | Missing Product request | Missing Product maps to HTTP 404 | Demonstrates gRPC `NOT_FOUND` to REST `404` mapping. |
| 14 | Missing Order request | Missing Order maps to HTTP 404 | Demonstrates Order not-found handling. |
| 15 | Order using a non-existent Product | Product dependency failure returns HTTP 404 | Proves Order service calls Product service over gRPC and does not store an invalid order. |
| 16 | Gateway error response examples | gRPC errors are mapped to HTTP status codes | Capture `400`, `404`, `503`, `504`, and `500` examples if available. |
| 17 | Product and Order database volumes | Database-per-service isolation | Show `ecommerce-product-db` and `ecommerce-order-db` as separate Compose volumes. |
| 18 | Product database contents | Product service owns the Product table | Show the Product SQLite file/table through a database viewer or service logs. |
| 19 | Order database contents | Order service owns the Order table | Show the Order SQLite file/table separately from Product data. |
| 20 | Order logs during Product outage | Retry attempts and exponential backoff | Capture `[PRODUCT RETRY]` logs while Product service is stopped. |
| 21 | Circuit breaker opening log | Circuit breaker changes from closed to open | Capture `[CIRCUIT BREAKER] product-service: closed -> open`. |
| 22 | Gateway request while Product is stopped | Stopped Product service returns HTTP 503 | Capture Postman response proving the request does not hang or crash. |
| 23 | Product service restart and first probe | Circuit breaker enters half-open state | After the breaker timeout, capture the half-open probe behavior/log. |
| 24 | Successful request after restart | Circuit breaker closes after recovery | Capture `[CIRCUIT BREAKER] product-service: half-open -> closed`. |
| 25 | Product Dockerfile | Product image uses a multi-stage build | Show the Go builder stage and small Alpine runtime stage. |
| 26 | Order Dockerfile | Order image uses a multi-stage build | Show the Go builder stage and small Alpine runtime stage. |
| 27 | Gateway Dockerfile | Gateway image uses a multi-stage build | Show the Go builder stage and small Alpine runtime stage. |
| 28 | Protobuf contracts and generated files | gRPC contracts define both service APIs | Show `proto/product.proto`, `proto/order.proto`, and generated Go files. |
| 29 | Architecture diagram in README | Final system architecture | Show the Mermaid diagram with Postman, gateway, both services, gRPC, and separate databases. |

## Recommended evidence commands

```bash
docker compose ps
docker compose logs order-service
docker volume ls | grep ecommerce
docker compose stop product-service
docker compose start product-service
```

For the resilience screenshots, keep the Order service log stream open:

```bash
docker compose logs -f order-service
```
