package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

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
