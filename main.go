package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
)

// run api
func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// read command line parameters
	addr := flag.String("addr", ":8080", "identify address that server listen to, by default :8080")
	mq := flag.String("queue", "amqp://guest:guest@localhost:8099/", "rabbitmq connect string")
	dsn := flag.String(
		"dsn",
		"user:my-secret-pw@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=True&loc=Local",
		"mysql connect string")
	couriers := flag.String("couriers", "http://localhost:8081/", "the url of couriers, split by single space")
	flag.Parse()

	courierArr := strings.Split(*couriers, " ")

	server := CreateServer(*addr, *mq, *dsn, courierArr)

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
