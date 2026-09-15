package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/liddiard-research/go-dbscope/feature/stdlib"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		Logger.Errorw("example failed", "error", err)
	}
}
func run(ctx context.Context) error {
	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("dbscope"),
		postgres.WithUsername("dbscope"),
		postgres.WithPassword("dbscope-test"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres: %w", err)
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			Logger.Errorw("failed to terminate postgres", "error", err)
		}
	}()
	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get postgres connection string: %w", err)
	}

	database, err := sql.Open("pgx", connectionString)
	if err != nil {
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			Logger.Errorw("close database", "error", err)
		}
	}()
	if err := database.PingContext(ctx); err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS orders (id BIGSERIAL PRIMARY KEY, customer_id BIGINT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventory (product_id BIGINT PRIMARY KEY, available INTEGER NOT NULL)`,
		`INSERT INTO inventory (product_id, available) VALUES (1, 10) ON CONFLICT (product_id) DO NOTHING`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	scope := stdlib.New(database)
	app := NewApp(scope, &OrderRepository{scope: scope}, &InventoryRepository{scope: scope})
	return app.run(ctx)
}
