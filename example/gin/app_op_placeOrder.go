package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PlaceOrderRequest struct {
	CustomerId int64 `json:"customerId" binding:"required,gt=0"`
	ProductId  int64 `json:"productId" binding:"required,gt=0"`
	Quantity   int   `json:"quantity" binding:"required,gt=0"`
}

func (app *App) PlaceOrder(ctx *gin.Context) {
	var request PlaceOrderRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.Abort()
		_ = ctx.Error(&requestError{message: "customerId, productId and quantity must be positive numbers"})
		return
	}
	orderId, err := app.orders.Create(ctx.Request.Context(), request.CustomerId)
	if err != nil {
		ctx.Abort()
		_ = ctx.Error(err)
		return
	}
	if err := app.inventory.Reserve(ctx.Request.Context(), request.ProductId, request.Quantity); err != nil {
		ctx.Abort()
		_ = ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"orderId": orderId})
}
