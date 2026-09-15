package main

import (
	"context"
	"errors"

	"github.com/liddiard-research/go-dbscope"

	pgxscope "github.com/liddiard-research/go-dbscope/feature/pgx"
)

type App struct {
	scope     *pgxscope.Scope
	orders    *OrderRepository
	inventory *InventoryRepository
}

func NewApp(
	scope *pgxscope.Scope,
	orders *OrderRepository,
	inventory *InventoryRepository,
) *App {
	return &App{
		scope:     scope,
		orders:    orders,
		inventory: inventory,
	}
}

func (receiver *App) run(ctx context.Context) error {
	available, err := receiver.inventory.Available(ctx, 1)
	if err != nil {
		return err
	}
	Logger.Infof("before transaction: %d available", available)
	if err := receiver.placeOrder(ctx, placeOrderRequest{customerId: 42, productId: 1, quantity: 2}); err != nil {
		return err
	}

	available, err = receiver.inventory.Available(ctx, 1)
	if err != nil {
		return err
	}
	Logger.Infof("after outer commit: %d available", available)
	return nil
}

func (receiver *App) placeOrder(ctx context.Context, request placeOrderRequest) error {
	txCtx, tx, err := receiver.scope.WithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(txCtx); err != nil && !errors.Is(err, dbscope.ErrTransactionClosed) {
			Logger.Errorf("failed to roll back transaction: %v", err)
		}
	}()

	orderId, err := receiver.orders.Create(txCtx, request.customerId)
	if err != nil {
		return err
	}
	if err := receiver.inventory.Reserve(txCtx, request.productId, request.quantity); err != nil {
		return err
	}
	available, err := receiver.inventory.Available(txCtx, request.productId)
	if err != nil {
		return err
	}
	Logger.Infof("order %d inside transaction: %d available", orderId, available)
	if err := receiver.tryAdditionalReservation(txCtx, request.productId, 5); err != nil {
		return err
	}
	available, err = receiver.inventory.Available(txCtx, request.productId)
	if err != nil {
		return err
	}
	Logger.Infof("after nested rollback: %d available", available)
	return tx.Commit(txCtx)
}

func (receiver *App) tryAdditionalReservation(ctx context.Context, productId int64, quantity int) error {
	nestedCtx, tx, err := receiver.scope.WithNestedTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(nestedCtx); err != nil && !errors.Is(err, dbscope.ErrTransactionClosed) {
			Logger.Errorf("failed to roll back nested transaction: %v", err)
		}
	}()
	if err := receiver.inventory.Reserve(nestedCtx, productId, quantity); err != nil {
		return err
	}
	available, err := receiver.inventory.Available(nestedCtx, productId)
	if err != nil {
		return err
	}
	Logger.Infof("inside nested transaction: %d available", available)
	return tx.Rollback(nestedCtx)
}
