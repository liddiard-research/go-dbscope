package stdlib

import (
	"context"
	"database/sql"

	"github.com/liddiard-research/go-dbscope"
)

type Connection interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Scope = dbscope.Scope[*sql.Tx, sql.TxOptions, Connection]

var _ Connection = (*sql.DB)(nil)
var _ Connection = (*sql.Tx)(nil)

type adapter struct {
	dbscope.UnimplementedAdapter[*sql.Tx, sql.TxOptions, Connection]
	database *sql.DB
}

func New(database *sql.DB) *Scope {
	return dbscope.New[*sql.Tx, sql.TxOptions, Connection](adapter{database: database})
}

func (adapter adapter) Connection(_ context.Context) (Connection, error) {
	return adapter.database, nil
}

func (adapter adapter) Begin(ctx context.Context) (*sql.Tx, error) {
	return adapter.database.BeginTx(ctx, nil)
}

func (adapter adapter) BeginOptions(ctx context.Context, options sql.TxOptions) (*sql.Tx, error) {
	return adapter.database.BeginTx(ctx, &options)
}

func (adapter adapter) Commit(_ context.Context, transaction *sql.Tx) error {
	return transaction.Commit()
}

func (adapter adapter) Rollback(_ context.Context, transaction *sql.Tx) error {
	return transaction.Rollback()
}
