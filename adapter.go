package dbscope

import "context"

type Adapter[T, O, C any] interface {
	TransactionLifecycle[T]

	Connection(context.Context) (C, error)
	Begin(context.Context) (T, error)
	BeginOptions(context.Context, O) (T, error)
	BeginNested(context.Context, T) (T, error)
}

type UnimplementedAdapter[T, O, C any] struct{}

func (UnimplementedAdapter[T, O, C]) Connection(context.Context) (C, error) {
	var zero C
	return zero, ErrNotImplemented
}

func (UnimplementedAdapter[T, O, C]) Begin(context.Context) (T, error) {
	var zero T
	return zero, ErrNotImplemented
}

func (UnimplementedAdapter[T, O, C]) BeginOptions(context.Context, O) (T, error) {
	var zero T
	return zero, ErrNotImplemented
}

func (UnimplementedAdapter[T, O, C]) BeginNested(context.Context, T) (T, error) {
	var zero T
	return zero, ErrNestedTransactionsUnsupported
}

func (UnimplementedAdapter[T, O, C]) Commit(context.Context, T) error {
	return ErrNotImplemented
}

func (UnimplementedAdapter[T, O, C]) Rollback(context.Context, T) error {
	return ErrNotImplemented
}
