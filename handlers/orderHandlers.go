package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	QueueManager tools.IQueueManager

	Ctx context.Context
}

func (s *ServerHandler) ReceiveOrder(ctx *gin.Context) {
	var requestJson []*types.Order
	if err := ctx.BindJSON(&requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	for i := 0; i < len(requestJson); i++ {
		fmt.Printf("struct: %v\n", *requestJson[i])
	}
	ctx.String(http.StatusOK, "received")
}

func (s *ServerHandler) ReceiveOrderFIFO(ctx *gin.Context) {
	var requestJson []*types.Order
	if err := ctx.BindJSON(&requestJson); err != nil {
		ctx.String(http.StatusBadRequest, err.Error())
		return
	}
	for i := 0; i < len(requestJson); i++ {
		fmt.Printf("send struct to queue: %v\n", *requestJson[i])
		err := s.QueueManager.Send(requestJson[i])
		if err != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("send message error: %v", err))
			return
		}
	}
	ctx.String(http.StatusOK, "received")
}
