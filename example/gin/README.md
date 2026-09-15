# Gin example

A demonstration of implicit transaction creation at the request level with Gin and `dbscope`.

Middleware implicitly creates a transaction for each mutating request and propagates it through the request context. Handlers and repositories participate without explicitly creating or resolving transactions. The middleware commits on success and rolls back on failure.

The example creates an order and reserves inventory through two repositories sharing one scope. Transaction boundaries are handled centrally, keeping handlers focused on application logic and repositories focused on database access.

## Run

Requires Go 1.27 or later and Docker. From this directory:

```bash
go run .
```

The application starts PostgreSQL and listens on port 8080.

```bash
curl -i http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"customerId":42,"productId":1,"quantity":2}'
```

The endpoint returns the created `orderId`. A failed inventory reservation rolls back the order as well.

Responses are not buffered: commit occurs after the handler returns, and any commit failure after a response has been sent is logged.
