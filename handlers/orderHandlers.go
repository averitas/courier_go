package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/averitas/courier_go/services"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	OrderService *services.OrderService

	Ctx context.Context
}

func (s *ServerHandler) ReceiveOrder(ctx *gin.Context) {
	var requestJson []*types.Order
	fmt.Printf("Start to find a random courier\n")
	if err := ctx.BindJSON(&requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	for i := 0; i < len(requestJson); i++ {
		fmt.Printf("save order: %v to DB\n", *requestJson[i])
		requestJson[i].OrderType = types.OrderTypeMatch
		err := s.OrderService.SaveOrder(requestJson[i])
		if err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("save order to db error: %v", err))
			return
		}
		fmt.Printf("Start to call random courier api: %v\n", *requestJson[i])
		err = s.OrderService.CallRandomCourierAPI(requestJson[i])
		if err != nil {
			ctx.String(http.StatusInternalServerError, "error with message: "+err.Error())
			return
		}
	}
	ctx.String(http.StatusAccepted, "received")
}

func (s *ServerHandler) ReceiveOrderFIFO(ctx *gin.Context) {
	var requestJson []*types.Order
	if err := ctx.BindJSON(&requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	for i := 0; i < len(requestJson); i++ {
		fmt.Printf("save order: %v to DB\n", *requestJson[i])
		requestJson[i].OrderType = types.OrderTypeFIFO
		err := s.OrderService.SaveOrder(requestJson[i])
		if err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("save order to db error: %v", err))
			return
		}

		fmt.Printf("send message to queue: %v\n", *requestJson[i])
		err = s.OrderService.SendOrderMessage(requestJson[i])
		if err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("send message error: %v", err))
			return
		}
	}
	ctx.String(http.StatusAccepted, "received")
}
