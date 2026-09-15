package dbscope

import (
	"context"
	"fmt"
	"sync"
)

type TransactionLifecycle[T any] interface {
	Commit(context.Context, T) error
	Rollback(context.Context, T) error
}

type joinedLifecycle[T any] struct{}

func (joinedLifecycle[T]) Commit(context.Context, T) error   { return nil }
func (joinedLifecycle[T]) Rollback(context.Context, T) error { return nil }

type Tx[T any] struct {
	transaction T
	lifecycle   TransactionLifecycle[T]

	mu     sync.RWMutex
	closed bool
}

func newTx[T any](transaction T, lifecycle TransactionLifecycle[T]) *Tx[T] {
	return &Tx[T]{transaction: transaction, lifecycle: lifecycle}
}

func (tx *Tx[T]) Transaction() T { return tx.transaction }

func (tx *Tx[T]) Commit(ctx context.Context) error {
	if !tx.close() {
		return ErrTransactionClosed
	}
	if err := tx.lifecycle.Commit(ctx, tx.transaction); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (tx *Tx[T]) Rollback(ctx context.Context) error {
	if !tx.close() {
		return ErrTransactionClosed
	}
	if err := tx.lifecycle.Rollback(ctx, tx.transaction); err != nil {
		return fmt.Errorf("rollback transaction: %w", err)
	}
	return nil
}

func (tx *Tx[T]) close() bool {
	tx.mu.Lock()
	defer tx.mu.Unlock()
	if tx.closed {
		return false
	}
	tx.closed = true
	return true
}

func (tx *Tx[T]) Closed() bool {
	tx.mu.RLock()
	defer tx.mu.RUnlock()
	return tx.closed
}
