package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/averitas/courier_go/db"
	"github.com/averitas/courier_go/handlers"
	"github.com/averitas/courier_go/repository"
	"github.com/averitas/courier_go/services"
	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/tools/logger"
	"github.com/averitas/courier_go/types"
	"github.com/gin-gonic/gin"
)

type Server struct {
	queueManager *tools.RabbitMqManager
	handler      *handlers.CourierHandler
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

	// start queue receiver
	go func() {
		defer func() {
			s.waitGroup.Done()
		}()
		err := s.queueManager.StartReceiver(ctx, func(b []byte) error {
			logger.InfoLogger.Printf("Received message: %s\n", string(b))
			return s.handler.HandleMessage(b)
		})
		if err != nil {
			logger.ErrorLogger.Printf("queue receiver abort with error %v\n", err)
			panic("queue receiver error " + err.Error())
		}
	}()

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := s.serverInst.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorLogger.Printf("Could not start listener %v\n", err)
		}
	}()
	<-ctx.Done()
	ctx1, cancel := context.WithTimeout(ctx, 2*time.Second)
	if err := s.serverInst.Shutdown(ctx1); err != nil {
		logger.ErrorLogger.Printf("Server force shutdown with error: %v\n", err)
	}
	cancel()

	// done with api server shutdown
	s.waitGroup.Done()

	// wait api server and queue sender
	s.waitGroup.Wait()
	logger.InfoLogger.Println("Server stopped")
}

func CreateServer(addr, queueConnString, dsn string) *Server {
	var router = gin.Default()

	// init thrid party tools managers
	queueManager := &tools.RabbitMqManager{
		QueueName:  types.QueueName,
		ConnString: queueConnString,
	}

	// init db
	db.InitDb(dsn)

	// init Service
	orderService := &services.OrderService{
		Repo:         &repository.OrderRepo{},
		HttpClient:   http.DefaultClient,
		QueueManager: queueManager,
		CouriersUrl:  make([]string, 0),
	}

	// init api server controller
	handler := &handlers.CourierHandler{
		OrderService: orderService,
	}

	// init api routers
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

func configureRouters(gEngin *gin.Engine, handler *handlers.CourierHandler) {
	// test api
	gEngin.GET("ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	// config api
	var api = gEngin.Group("/api")
	api.POST("sendOrder", handler.SendOrder)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	// read command line parameters
	addr := flag.String("addr", ":8081", "identify address that server listen to, by default :8081")
	mq := flag.String("queue", "amqp://guest:guest@localhost:8099/", "rabbitmq connect string")
	dsn := flag.String(
		"dsn",
		"user:my-secret-pw@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		"mysql connect string")
	flag.Parse()

	logger.InfoLogger.Printf("Start with port: %s\n", *addr)

	server := CreateServer(*addr, *mq, *dsn)

	// catch ctrl + c
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			logger.InfoLogger.Println("Received int signal, try to shutdown gracefully")
			cancel()
			break
		}
	}()

	server.StartAndWait(ctx)

	<-ctx.Done()
}
