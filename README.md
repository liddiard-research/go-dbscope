# dbscope

Typed transaction propagation for Go repositories.

`dbscope` propagates concrete database transactions through `context.Context` and resolves the appropriate connection view for repository code. The root module is database/library agnostic; feature modules adapt concrete database libraries.

## Table of contents

- [Why dbscope](#why-dbscope)
- [Installation](#installation)
- [Usage](#usage)
- [Transactions](#transactions)
- [Nested transactions](#nested-transactions)
- [Supported integrations](#supported-integrations)
- [Examples](#examples)
- [Licence](#licence)

## Why dbscope

Design patterns like repositories isolate database access very well. Transactions make several database operations atomic whereby their changes are committed together or discarded together. Composing repositories into an atomic multi step operation should not require each repository to manage transaction boundaries.

Without a shared propagation mechanism, repositories often need separate paths for a database connection and a transaction, despite their similar query operations. Transaction handling spreads through methods whose responsibility is data access. `dbscope` resolves that duplication through a shared connection view:

```go
connection, err := scope.Connection(ctx)
```

Go contexts carry request scoped values, cancellation and deadlines across call boundaries. `dbscope` uses a derived context to carry the active transaction, so the repository remains transaction agnostic. Controlling of the boundary moves further to where it makes sense to assemble and passes a context downstream:

```text
normal context      -> base connection view
transaction context -> active transaction view
```

## Installation

Requires Go 1.27.0 or later.

```bash
go get github.com/liddiard-research/go-dbscope@latest
```

Install the integration for your database library:

```bash
go get github.com/liddiard-research/go-dbscope/feature/pgx@latest
go get github.com/liddiard-research/go-dbscope/feature/stdlib@latest
```

## Usage

The following example uses the `feature/pgx` integration.

Import `github.com/liddiard-research/go-dbscope/feature/pgx` as `pgxscope`. A repository uses the scope and resolves its connection for each operation:

```go
type UserRepository struct {
    scope *pgxscope.Scope
}

func (repository *UserRepository) Create(
    ctx context.Context,
    name string,
) error {
    connection, err := repository.scope.Connection(ctx)
    if err != nil {
        return err
    }

    _, err = connection.Exec(
        ctx,
        "INSERT INTO users (name) VALUES ($1)",
        name,
    )

    return err
}
```

Construct one scope from your existing pool and share it between repositories that participate in the same transaction:

```go
scope := pgxscope.New(pool)
repository := &UserRepository{scope: scope}
```

In the pgx integration, `Connection(ctx)` resolves either the application’s pool or the active pgx transaction through their shared connection interface. The application owns the pool and its lifecycle. Resolving a connection does not begin a transaction.

## Transactions

Use an explicit boundary for operations that must succeed together. Here, `users` and `audit` share the same scope:

```go
ctx, tx, err := scope.WithTx(ctx)
if err != nil {
    return err
}

defer tx.Rollback(ctx)

if err := users.Create(ctx, "Jack"); err != nil {
    return err
}

if err := audit.Record(ctx, "user created"); err != nil {
    return err
}

return tx.Commit(ctx)
```

`WithTx` creates a root transaction when none exists. If the context already carries a transaction for the same scope, it joins that transaction and returns the original context. Pass the returned context to repositories and downstream services; the transaction owner commits or rolls back the boundary.

A joined handle does not commit or roll back its parent transaction. Joined rollback does not undo statements already executed in that transaction.

Use `WithTxOptions(ctx, options)` to create a root transaction with native driver options. Options are rejected when a transaction is already active. `GetTx(ctx)` only retrieves an existing concrete transaction; it never creates one. Do not reuse a transaction context after its owning boundary has ended.

### API summary

| Method | Purpose |
| --- | --- |
| `Connection(ctx)` | Return the base connection or active transaction through the integration’s shared connection interface. |
| `GetTx(ctx)` | Retrieve the concrete transaction already propagated through the context. |
| `WithTx(ctx)` | Join an existing transaction or create a root transaction. |
| `WithTxOptions(ctx, options)` | Create a root transaction with native driver options. |
| `WithNestedTx(ctx)` | Create a root or nested transaction boundary. |
| `Within(ctx, fn)` | Convenience wrapper around `WithTx`, rollback and commit. |

## Nested transactions

Use a nested boundary when an operation needs its own rollback boundary within a larger transaction:

```go
nestedCtx, nested, err := scope.WithNestedTx(txCtx)
if err != nil {
    return err
}

defer nested.Rollback(nestedCtx)

if err := inventory.TryReserve(nestedCtx, productId); err != nil {
    return err
}

return nested.Commit(nestedCtx)
```

pgx implements nested transactions using savepoints. `database/sql` has no generic native nested transaction API, so `feature/stdlib` returns `ErrNestedTransactionsUnsupported` when nesting inside an existing transaction. Without an active transaction, `WithNestedTx` creates a root boundary.

## Supported integrations

| Integration | Package | Nested transactions |
| --- | --- | --- |
| pgx | `feature/pgx` | Yes, using native savepoint-backed transactions |
| `database/sql` | `feature/stdlib` | No generic nested transaction API |

The root module defines transaction propagation and lifecycle behaviour through `dbscope.Scope[T, O, C]`. Integrations are isolated under `feature/` and adapt native transaction, options and shared connection types to that same generic model. Applications select the feature package appropriate for their database access layer; repositories depend on its connection interface. Applications using `database/sql` can use the `feature/stdlib` integration. Custom integrations implement `dbscope.Adapter[T, O, C]`, may embed `dbscope.UnimplementedAdapter[T, O, C]`, and construct a scope with `dbscope.New(adapter)`.

## Examples

Docker is required to run the examples; each starts and cleans up its own PostgreSQL container.

- [`example/pgx`](example/pgx) shows usage with the pgx integration.
- [`example/stdlib`](example/stdlib) shows usage with the `database/sql` integration.
- [`example/gin`](example/gin) shows a Gin application using a transaction for each mutating request.

## Licence

Released under the [MIT Licence](LICENSE).

Copyright © 2026 Liddiard Research Limited.
