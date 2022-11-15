package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/averitas/courier_go/handlers"
	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type Server struct {
	queueManager *tools.RabbitMqManager
	handler      *handlers.ServerHandler
	serverInst   *http.Server

	waitGroup *sync.WaitGroup
}

func (s *Server) StartAndWait(ctx context.Context) {
	err := s.queueManager.Init()
	if err != nil {
		panic(fmt.Sprintf("init queue error: %v", err))
	}

	// add two wait group: 1. api server, 2. background queue sender
	s.waitGroup.Add(2)

	go func() {
		defer func() {
			s.waitGroup.Done()
		}()
		s.queueManager.StartSender(ctx)
	}()

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := s.serverInst.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Could not start listener %v\n", err)
		}
	}()
	<-ctx.Done()
	ctx1, cancel := context.WithTimeout(ctx, 2*time.Second)
	if err := s.serverInst.Shutdown(ctx1); err != nil {
		fmt.Printf("Server force shutdown with error: %v\n", err)
	}
	cancel()

	// done with api server shutdown
	s.waitGroup.Done()

	// wait api server and queue sender
	s.waitGroup.Wait()
	fmt.Println("Server stopped")
}

func CreateServer(addr string, queueConnString string) *Server {
	var router = gin.Default()

	queueManager := &tools.RabbitMqManager{
		QueueName:  types.QueueName,
		ConnString: queueConnString,
	}

	handler := &handlers.ServerHandler{
		QueueManager: queueManager,
	}

	configureRouters(router, handler)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	return &Server{
		queueManager: queueManager,
		serverInst:   server,
		handler:      handler,
		waitGroup:    &sync.WaitGroup{},
	}
}

func configureRouters(gEngin *gin.Engine, handler *handlers.ServerHandler) {
	// test api
	gEngin.GET("ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	// config api
	var api = gEngin.Group("/api")
	api.POST("sendOrder/random", handler.ReceiveOrder)
	api.POST("sendOrder/fifo", handler.ReceiveOrderFIFO)
}

// run api
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	server := CreateServer(":8080", "amqp://guest:guest@localhost:8099/")

	// catch ctrl + c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Println("Received int signal, try to shutdown gracefully")
			cancel()
			break
		}
	}()

	server.StartAndWait(ctx)

	<-ctx.Done()
}
