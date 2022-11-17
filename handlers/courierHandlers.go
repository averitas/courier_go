package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/averitas/courier_go/services"
	"github.com/averitas/courier_go/tools/logger"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

// handler instance
type CourierHandler struct {
	OrderService *services.OrderService

	Ctx context.Context
}

// @description http handler that receive message from
// apiserver "matched" type order
// @param ctx *gin.Context
// @return
func (c *CourierHandler) SendOrder(ctx *gin.Context) {
	var requestJson *types.Order = &types.Order{}
	if err := ctx.BindJSON(requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	logger.InfoLogger.Printf("Received order: [%v]\n", *requestJson)

	orderModel, err := c.OrderService.GetOrderModel(requestJson)
	if err != nil {
		logger.ErrorLogger.Printf("received invalid order: %v\n", *requestJson)
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	go func() {
		err := c.OrderService.WaitUntilOrderCooked(orderModel)
		if err != nil {
			logger.ErrorLogger.Printf("order :[%s] cook error :%v\n", orderModel.OrderId, err)
		}
	}()
	ctx.Status(http.StatusAccepted)
}

// @description Ths function is used in queue receiver handler.
// it deserilize message, start a goroutine wait dish is ready,
// then set its status to finished.
// @param b []byte message body
// @return error
func (c *CourierHandler) HandleMessage(b []byte) error {
	var requestJson *types.Order = &types.Order{}
	err := json.Unmarshal(b, &requestJson)
	if err != nil {
		return fmt.Errorf("unmarshal message: [%s], error: %v", string(b), err)
	}
	logger.InfoLogger.Printf("Start to handler order: [%v]\n", *requestJson)

	orderModel, err := c.OrderService.GetOrderModel(requestJson)
	if err != nil {
		logger.ErrorLogger.Printf("[ERROR] received invalid order: %v, error: %v\n", *requestJson, err)
		return err
	}

	// start to cook
	go func() {
		err := c.OrderService.WaitUntilOrderCooked(orderModel)
		if err != nil {
			logger.ErrorLogger.Printf("[ERROR] Order :[%v] cook error :%v\n", orderModel, err)
		}
	}()

	return nil
}
