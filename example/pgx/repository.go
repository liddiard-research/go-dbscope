package main

import (
	"context"
	"fmt"

	"github.com/liddiard-research/go-dbscope/feature/pgx"
)

type OrderRepository struct{ scope *pgxscope.Scope }

func (repository *OrderRepository) Create(ctx context.Context, customerId int64) (int64, error) {
	connection, err := repository.scope.Connection(ctx)
	if err != nil {
		return 0, err
	}
	var orderId int64
	err = connection.QueryRow(ctx, "INSERT INTO orders (customer_id) VALUES ($1) RETURNING id", customerId).Scan(&orderId)
	return orderId, err
}

type InventoryRepository struct{ scope *pgxscope.Scope }

func (repository *InventoryRepository) Available(ctx context.Context, productId int64) (int, error) {
	connection, err := repository.scope.Connection(ctx)
	if err != nil {
		return 0, err
	}
	var available int
	err = connection.QueryRow(ctx, "SELECT available FROM inventory WHERE product_id = $1", productId).Scan(&available)
	return available, err
}

func (repository *InventoryRepository) Reserve(ctx context.Context, productId int64, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("reservation quantity must be positive")
	}
	connection, err := repository.scope.Connection(ctx)
	if err != nil {
		return err
	}
	result, err := connection.Exec(ctx,
		"UPDATE inventory SET available = available - $2 WHERE product_id = $1 AND available >= $2",
		productId,
		quantity,
	)
	if err != nil {
		return err
	}
	affected := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("product %d has insufficient inventory or does not exist", productId)
	}
	return nil
}
