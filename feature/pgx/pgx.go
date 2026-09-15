package pgxscope

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liddiard-research/go-dbscope"
)

type Connection interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Scope = dbscope.Scope[pgx.Tx, pgx.TxOptions, Connection]

var _ Connection = (*pgxpool.Pool)(nil)
var _ Connection = (pgx.Tx)(nil)

type adapter struct {
	dbscope.UnimplementedAdapter[pgx.Tx, pgx.TxOptions, Connection]
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Scope {
	return dbscope.New[pgx.Tx, pgx.TxOptions, Connection](adapter{pool: pool})
}

func (adapter adapter) Connection(_ context.Context) (Connection, error) {
	return adapter.pool, nil
}

func (adapter adapter) Begin(ctx context.Context) (pgx.Tx, error) {
	return adapter.pool.Begin(ctx)
}

func (adapter adapter) BeginOptions(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	return adapter.pool.BeginTx(ctx, options)
}

func (adapter adapter) BeginNested(ctx context.Context, transaction pgx.Tx) (pgx.Tx, error) {
	return transaction.Begin(ctx)
}

func (adapter adapter) Commit(ctx context.Context, transaction pgx.Tx) error {
	return transaction.Commit(ctx)
}

func (adapter adapter) Rollback(ctx context.Context, transaction pgx.Tx) error {
	return transaction.Rollback(ctx)
}
