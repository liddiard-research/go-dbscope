package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liddiard-research/go-dbscope/feature/pgx"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type placeOrderRequest struct {
	customerId int64
	productId  int64
	quantity   int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, cleanup, err := makeApp(ctx)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := cleanup(context.Background()); err != nil {
			Logger.Errorf("failed to clean up app: %v", err)
		}
	}()

	if err := app.run(ctx); err != nil {
		panic(err)
	}
}

func makeApp(ctx context.Context) (*App, func(ctx context.Context) error, error) {
	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("dbscope"),
		postgres.WithUsername("dbscope"),
		postgres.WithPassword("dbscope-test"),
		postgres.BasicWaitStrategies(),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	var pool *pgxpool.Pool
	cleanup := func(ctx context.Context) error {
		if pool != nil {
			pool.Close()
		}
		if err := container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate postgres: %w", err)
		}
		return nil
	}
	ready := false
	defer func() {
		if !ready {
			if err := cleanup(context.Background()); err != nil {
				Logger.Errorf("failed to clean up app after setup failure: %v", err)
			}
		}
	}()

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get postgres connection string: %w", err)
	}
	pool, err = pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	scope := pgxscope.New(pool)

	if err := migrate(ctx, pool); err != nil {
		return nil, nil, fmt.Errorf("failed to migrate: %w", err)
	}

	ready = true

	return NewApp(
		scope,
		&OrderRepository{
			scope: scope,
		},
		&InventoryRepository{
			scope: scope,
		},
	), cleanup, nil
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS orders (id BIGSERIAL PRIMARY KEY, customer_id BIGINT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventory (product_id BIGINT PRIMARY KEY, available INTEGER NOT NULL)`,
		`INSERT INTO inventory (product_id, available) VALUES (1, 10) ON CONFLICT (product_id) DO UPDATE SET available = EXCLUDED.available`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}
	return nil
}
