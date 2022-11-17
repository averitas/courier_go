package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/averitas/courier_go/services"
	"github.com/averitas/courier_go/tools/logger"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	OrderService *services.OrderService

	Ctx context.Context
}

// @description http handler that user can call it
// to send order with "Matched" dispatch type
// @param ctx *gin.Context
// @return
func (s *ServerHandler) ReceiveOrder(ctx *gin.Context) {
	var requestJson []*types.Order
	logger.InfoLogger.Printf("Start to find a random courier\n")
	retval := &types.Message{
		Code:    types.CodeSuccess,
		Message: "received",
	}
	if err := ctx.BindJSON(&requestJson); err != nil {
		retval.Code = types.CodeFailed
		retval.Message = fmt.Sprintf("input json format err: %v", err)
		ctx.JSON(http.StatusBadRequest, retval)
		return
	}
	for i := 0; i < len(requestJson); i++ {
		logger.InfoLogger.Printf("save order: %v to DB\n", *requestJson[i])
		requestJson[i].OrderType = types.OrderTypeMatch
		err := s.OrderService.SaveOrder(requestJson[i])
		if err != nil {
			retval.Code = types.CodeFailed
			retval.Message = fmt.Sprintf("save order to db error: %v", err)
			ctx.JSON(http.StatusInternalServerError, retval)
			return
		}
		logger.InfoLogger.Printf("Start to call random courier api: %v\n", *requestJson[i])
		err = s.OrderService.CallRandomCourierAPI(requestJson[i])
		if err != nil {
			retval.Code = types.CodeFailed
			retval.Message = fmt.Sprintf("save order to db error: %v", err)
			ctx.JSON(http.StatusInternalServerError, retval)
			return
		}
	}
	ctx.JSON(http.StatusAccepted, "received")
}

// @description http handler that user can call it
// to send order with "FIFO" dispatch type
// @param ctx *gin.Context
// @return
func (s *ServerHandler) ReceiveOrderFIFO(ctx *gin.Context) {
	var requestJson []*types.Order
	if err := ctx.BindJSON(&requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	retval := &types.Message{
		Code:    types.CodeSuccess,
		Message: "received",
	}
	for i := 0; i < len(requestJson); i++ {
		logger.InfoLogger.Printf("save order: %v to DB\n", *requestJson[i])
		requestJson[i].OrderType = types.OrderTypeFIFO
		err := s.OrderService.SaveOrder(requestJson[i])
		if err != nil {
			retval.Code = types.CodeFailed
			retval.Message = fmt.Sprintf("save order to db error: %v", err)
			ctx.JSON(http.StatusInternalServerError, retval)
			return
		}

		logger.InfoLogger.Printf("send message to queue: %v\n", *requestJson[i])
		err = s.OrderService.SendOrderMessage(requestJson[i])
		if err != nil {
			retval.Code = types.CodeFailed
			retval.Message = fmt.Sprintf("send message error: %v", err)
			ctx.JSON(http.StatusInternalServerError, retval)
			return
		}
	}

	ctx.JSON(http.StatusAccepted, retval)
}

// @description http handler that user can call it
// to retrieve average dispatch delay time(in milliseconds) of requested type
// example: GET http://127.0.0.1:8080/api/delay/fifo
// @param ctx *gin.Context
// @return
func (s *ServerHandler) QueryAverageDelay(ctx *gin.Context) {
	orderType := ctx.Param("orderType")
	retval := &types.Message{
		Code:    types.CodeSuccess,
		Message: "received",
	}
	if len(orderType) == 0 {
		retval.Code = types.CodeFailed
		retval.Message = fmt.Sprintf("order type [%s] is invalid, please use match or fifo", orderType)
		ctx.JSON(http.StatusBadRequest, retval)
	}

	average, err := s.OrderService.GetAverageDelayOfType(orderType)
	if err != nil {
		retval.Code = types.CodeFailed
		retval.Message = fmt.Sprintf("query average error: %v", err)
		ctx.JSON(http.StatusInternalServerError, retval)
		return
	}
	retval.Message = fmt.Sprintf("Average dispatch delay is %v", average*1000)
	ctx.JSON(http.StatusOK, retval)
}
