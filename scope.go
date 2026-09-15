package dbscope

import (
	"context"
	"errors"
	"fmt"
)

type Scope[T, O, C any] struct {
	adapter    Adapter[T, O, C]
	contextKey *scopeContextKey
}

func New[T, O, C any](adapter Adapter[T, O, C]) *Scope[T, O, C] {
	return &Scope[T, O, C]{adapter: adapter, contextKey: &scopeContextKey{}}
}

type scopeContextKey struct {
	marker byte
}

type transactionContextValue[T any] struct {
	transaction T
}

func (scope *Scope[T, O, C]) Connection(ctx context.Context) (C, error) {
	if tx, ok := scope.GetTx(ctx); ok {
		if connection, ok := any(tx).(C); ok {
			return connection, nil
		}
		var zero C
		return zero, ErrTransactionConnectionUnsupported
	}
	return scope.adapter.Connection(ctx)
}

func (scope *Scope[T, O, C]) GetTx(ctx context.Context) (T, bool) {
	value, ok := ctx.Value(scope.contextKey).(transactionContextValue[T])
	if !ok {
		var zero T
		return zero, false
	}
	return value.transaction, true
}

func (scope *Scope[T, O, C]) WithTx(ctx context.Context) (context.Context, *Tx[T], error) {
	if transaction, ok := scope.GetTx(ctx); ok {
		return ctx, newTx(transaction, joinedLifecycle[T]{}), nil
	}
	transaction, err := scope.adapter.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	return scope.withTransaction(ctx, transaction)
}

func (scope *Scope[T, O, C]) WithTxOptions(ctx context.Context, options O) (context.Context, *Tx[T], error) {
	if _, ok := scope.GetTx(ctx); ok {
		return nil, nil, ErrTransactionOptionsAlreadyActive
	}
	transaction, err := scope.adapter.BeginOptions(ctx, options)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	return scope.withTransaction(ctx, transaction)
}

func (scope *Scope[T, O, C]) WithNestedTx(ctx context.Context) (context.Context, *Tx[T], error) {
	parent, ok := scope.GetTx(ctx)
	if !ok {
		return scope.WithTx(ctx)
	}
	transaction, err := scope.adapter.BeginNested(ctx, parent)
	if err != nil {
		return nil, nil, fmt.Errorf("begin nested transaction: %w", err)
	}
	return scope.withTransaction(ctx, transaction)
}

func (scope *Scope[T, O, C]) withTransaction(ctx context.Context, transaction T) (context.Context, *Tx[T], error) {
	txCtx := context.WithValue(ctx, scope.contextKey, transactionContextValue[T]{transaction: transaction})
	return txCtx, newTx(transaction, scope.adapter), nil
}

func (scope *Scope[T, O, C]) Within(ctx context.Context, fn func(context.Context) error) error {
	txCtx, tx, err := scope.WithTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(txCtx)
	if callbackErr := fn(txCtx); callbackErr != nil {
		if rollbackErr := tx.Rollback(txCtx); rollbackErr != nil {
			return errors.Join(callbackErr, rollbackErr)
		}
		return callbackErr
	}
	return tx.Commit(txCtx)
}
