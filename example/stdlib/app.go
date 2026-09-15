package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/liddiard-research/go-dbscope"
	"github.com/liddiard-research/go-dbscope/feature/stdlib"
)

type App struct {
	scope     *stdlib.Scope
	orders    *OrderRepository
	inventory *InventoryRepository
}

func NewApp(
	scope *stdlib.Scope,
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
	Logger.Infow("before transaction", "available", available)
	if err := receiver.placeOrder(ctx, placeOrderRequest{customerId: 42, productId: 1, quantity: 2}); err != nil {
		return err
	}

	available, err = receiver.inventory.Available(ctx, 1)
	if err != nil {
		return err
	}
	Logger.Infow("after outer commit", "available", available)
	return nil
}

type placeOrderRequest struct {
	customerId int64
	productId  int64
	quantity   int
}

func (receiver *App) placeOrder(ctx context.Context, request placeOrderRequest) error {
	ctx, tx, err := receiver.scope.WithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, dbscope.ErrTransactionClosed) {
			Logger.Errorw("failed to roll back transaction", "error", err)
		}
	}()

	orderId, err := receiver.orders.Create(ctx, request.customerId)
	if err != nil {
		return err
	}
	if err := receiver.inventory.Reserve(ctx, request.productId, request.quantity); err != nil {
		return err
	}
	available, err := receiver.inventory.Available(ctx, request.productId)
	if err != nil {
		return err
	}
	Logger.Infow("inside transaction", "orderId", orderId, "available", available)
	_, _, err = receiver.scope.WithNestedTx(ctx)
	if !errors.Is(err, dbscope.ErrNestedTransactionsUnsupported) {
		return fmt.Errorf("expected nested transactions to be unsupported: %w", err)
	}
	Logger.Infow("nested transactions are unsupported by database/sql")
	return tx.Commit(ctx)
}
