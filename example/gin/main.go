package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liddiard-research/go-dbscope/feature/pgx"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		Logger.Errorw("server failed", "error", err)
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

	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}

	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS orders (id BIGSERIAL PRIMARY KEY, customer_id BIGINT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS inventory (product_id BIGINT PRIMARY KEY, available INTEGER NOT NULL)`,
		`INSERT INTO inventory (product_id, available) VALUES (1, 10) ON CONFLICT (product_id) DO NOTHING`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			return err
		}
	}

	scope := pgxscope.New(pool)
	app := NewApp(scope, &OrderRepository{scope: scope}, &InventoryRepository{scope: scope})
	router := gin.Default()
	router.Use(app.transactionMiddleware())
	router.POST("/orders", app.PlaceOrder)
	server := &http.Server{Addr: ":8080", Handler: router, ReadHeaderTimeout: 5 * time.Second}

	return server.ListenAndServe()
}
