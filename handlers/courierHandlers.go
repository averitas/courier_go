package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/averitas/courier_go/services"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type CourierHandler struct {
	OrderService *services.OrderService

	Ctx context.Context
}

func (c *CourierHandler) SendOrder(ctx *gin.Context) {
	var requestJson *types.Order = &types.Order{}
	if err := ctx.BindJSON(requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	fmt.Printf("Received order: [%v]\n", *requestJson)

	orderModel, err := c.OrderService.GetOrderModel(requestJson)
	if err != nil {
		fmt.Printf("[ERROR] received invalid order: %v\n", *requestJson)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	go func() {
		err := c.OrderService.WaitUntilOrderCooked(orderModel)
		if err != nil {
			fmt.Printf("[ERROR] Order :[%s] cook error :%v\n", orderModel.OrderId, err)
		}
	}()

	ctx.String(http.StatusAccepted, "received")
}

func (c *CourierHandler) HandleMessage(b []byte) error {
	var requestJson *types.Order = &types.Order{}
	err := json.Unmarshal(b, &requestJson)
	if err != nil {
		return fmt.Errorf("unmarshal message: [%s], error: %v", string(b), err)
	}
	fmt.Printf("Start to handler order: [%v]\n", *requestJson)

	orderModel, err := c.OrderService.GetOrderModel(requestJson)
	if err != nil {
		fmt.Printf("[ERROR] received invalid order: %v, error: %v\n", *requestJson, err)
		return err
	}

	// start to cook
	go func() {
		err := c.OrderService.WaitUntilOrderCooked(orderModel)
		if err != nil {
			fmt.Printf("[ERROR] Order :[%v] cook error :%v\n", orderModel, err)
		}
	}()

	return nil
}
