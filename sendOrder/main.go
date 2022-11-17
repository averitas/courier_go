package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/averitas/courier_go/types"
	"github.com/google/uuid"
	"github.com/splode/fname"
)

func main() {
	inputUrl := flag.String("url", "http://localhost:8080/", "default is http://localhost:8080/")
	testType := flag.String("type", "fifo", "value should be: match or fifo")
	flag.Parse()

	targetUrl, err := url.Parse(*inputUrl)
	if err != nil {
		fmt.Printf("configured url %v is invalid\n", *inputUrl)
	}
	if *testType == "match" {
		targetUrl.Path = path.Join(targetUrl.Path, "api", "sendOrder", "random")
	} else if *testType == "fifo" {
		targetUrl.Path = path.Join(targetUrl.Path, "api", "sendOrder", "fifo")
	} else {
		panic(fmt.Sprintf("test type [%s] is invalid", *testType))
	}

	// name generator
	gen := fname.NewGenerator()

	// loop to send two random order per second
	for {
		newName, _ := gen.Generate()
		newName2, _ := gen.Generate()
		orderArr := []*types.Order{
			{
				Id:       uuid.New().String(),
				Name:     newName,
				PrepTime: 3 + rand.Intn(13),
			},
			{
				Id:       uuid.New().String(),
				Name:     newName2,
				PrepTime: 3 + rand.Intn(13),
			},
		}
		orderMessage, err := json.Marshal(orderArr)
		if err != nil {
			panic(err)
		}

		req, err := http.NewRequest(http.MethodPost, targetUrl.String(), bytes.NewReader(orderMessage))
		if err != nil {
			fmt.Printf("err when SendOrderMessage generate http request to url [%s] error: %v\n", targetUrl.String(), err)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("err when SendOrderMessage call url [%s] error: %v\n", targetUrl.String(), err)
		}

		fmt.Printf("API result is: %v\n", res)

		time.Sleep(time.Second)
	}
}
