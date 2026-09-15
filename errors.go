package dbscope

import "errors"

var (
	ErrTransactionConnectionUnsupported = errors.New("transaction does not implement connection view")

	ErrNotImplemented = errors.New("operation not implemented")

	ErrTransactionClosed = errors.New("transaction is already closed")

	ErrNestedTransactionsUnsupported = errors.New("nested transactions are not supported")

	ErrTransactionOptionsAlreadyActive = errors.New("transaction options cannot be applied to an active transaction")
)
