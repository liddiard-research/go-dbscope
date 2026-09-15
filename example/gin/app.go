package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/liddiard-research/go-dbscope"
	pgxscope "github.com/liddiard-research/go-dbscope/feature/pgx"
)

type App struct {
	scope     *pgxscope.Scope
	orders    *OrderRepository
	inventory *InventoryRepository
}

func NewApp(scope *pgxscope.Scope, orders *OrderRepository, inventory *InventoryRepository) *App {
	return &App{scope: scope, orders: orders, inventory: inventory}
}

type requestError struct{ message string }

func (err *requestError) Error() string { return err.message }

func (app *App) transactionMiddleware() gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		switch ginCtx.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			ginCtx.Next()
			return
		}
		txCtx, tx, err := app.scope.WithTx(ginCtx.Request.Context())
		if err != nil {
			writeError(ginCtx, err)
			return
		}

		ginCtx.Request = ginCtx.Request.WithContext(txCtx)

		defer func() {
			if err := tx.Rollback(context.WithoutCancel(txCtx)); err != nil && !errors.Is(err, dbscope.ErrTransactionClosed) {
				Logger.Errorw("rollback request transaction", "error", err)
			}
		}()

		ginCtx.Next()
		if len(ginCtx.Errors) > 0 {
			writeError(ginCtx, ginCtx.Errors.Last().Err)
			return
		}
		if ginCtx.IsAborted() || ginCtx.Writer.Status() >= http.StatusBadRequest {
			return
		}
		if err := tx.Commit(txCtx); err != nil {
			writeError(ginCtx, err)
		}
	}
}

func writeError(ginCtx *gin.Context, err error) {
	ginCtx.Abort()
	if ginCtx.Writer.Written() {
		Logger.Errorw("request failed after response was sent", "error", err)
		return
	}
	if invalid, ok := errors.AsType[*requestError](err); ok {
		ginCtx.JSON(http.StatusBadRequest, gin.H{"error": invalid.message})
		return
	}
	Logger.Errorw("request failed", "error", err)
	ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
