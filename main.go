package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// run api
func main() {
	var router = gin.Default()

	configureRouters(router)

	router.Run(":8080")
	println("test!")
}

func configureRouters(gEngin *gin.Engine) {
	// test api
	gEngin.GET("ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	// config api
	var api = gEngin.Group("/api")
	api.POST("sendOrder", handlers.receiveOrder)
}
