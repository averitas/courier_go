package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(1)

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

	// init 3rd party rooles
	queueManager := &tools.RabbitMqManager{
		QueueName:  types.QueueName,
		ConnString: "amqp://guest:guest@localhost:8099/",
	}

	go func() {
		err := queueManager.StartReceiver(ctx, func(b []byte) error {
			fmt.Printf("Received message: %s\n", string(b))
			return nil
		})
		cancel()
		if err != nil {
			fmt.Printf("queue receiver abort with error %v\n", err)
			panic("queue receiver error " + err.Error())
		}
		wg.Done()
	}()

	fmt.Println("Worker started successfully")
	wg.Wait()
}
