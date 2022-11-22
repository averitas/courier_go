package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/averitas/courier_go/models"
	"github.com/averitas/courier_go/repository"
	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/tools/logger"
	"github.com/averitas/courier_go/types"
)

type OrderService struct {
	QueueManager tools.IQueueManager
	Repo         repository.IOrderRepo
	HttpClient   tools.HttpClient
	CouriersUrl  []string
}

// @description Save order to database with given order struct
// @param order *types.Order order received from api
// @return error
func (o *OrderService) SaveOrder(order *types.Order) error {
	orderModel := &models.OrderModel{
		Id:          order.Id,
		PrepTime:    order.PrepTime,
		Name:        order.Name,
		OrderStatus: models.OrderStarted,
		OrderType:   order.OrderType,
	}
	err := o.Repo.CreateOrder(orderModel)
	if err != nil {
		return fmt.Errorf("save order error: %v", err)
	}
	return nil
}

// @description Get the single latest order with 'id' in order struct
// @param order *types.Order order received from api
// @return error
func (o *OrderService) GetOrderModel(order *types.Order) (res *models.OrderModel, err error) {
	res, err = o.Repo.GetOrderById(order.Id)
	return
}

// @description This function simulate courier wait kitchen cooking and
// finish this order. It will wait PrepTime seconds, then set status to finished
// @param model *models.OrderModel model retrieved from database
// @return error
func (o *OrderService) WaitUntilOrderCooked(model *models.OrderModel) (err error) {
	defer func() {
		if rcy := recover(); rcy != nil {
			err = fmt.Errorf("handler error: %v\n!panic: %v", err, rcy)
		}
	}()

	// set order status to cooking
	model.OrderStatus = models.OrderCooking
	err = o.Repo.SaveModel(model)
	if err != nil {
		return fmt.Errorf("order set status to cooking err: %v", err)
	}
	logger.InfoLogger.Printf("Order [%s] started cooking\n", model.OrderId)

	// wait order
	time.Sleep(time.Duration(model.PrepTime) * time.Second)

	// order is done, move status to finished
	model.OrderStatus = models.OrderFinished
	err = o.Repo.SaveModel(model)
	if err != nil {
		return fmt.Errorf("order set status to finished err: %v", err)
	}
	logger.InfoLogger.Printf("Order [%s] done!\n", model.OrderId)
	return nil
}

// @description This function send order message to queue
// @param order *types.Order order received from api
// @return error
func (o *OrderService) SendOrderMessage(order *types.Order) error {
	err := o.QueueManager.Send(order)
	if err != nil {
		return fmt.Errorf("send message error: %v", err)
	}
	return nil
}

// @description This function send order message to courier
// by call courier API directly. It will choose a random courier
// to send message
// @param order *types.Order order received from api
// @return error
func (o *OrderService) CallRandomCourierAPI(order *types.Order) error {
	if len(o.CouriersUrl) < 1 {
		return fmt.Errorf("please configure courier url first")
	}

	urlIndex := rand.Intn(len(o.CouriersUrl))
	targetUrl, err := url.Parse(o.CouriersUrl[urlIndex])
	if err != nil {
		return fmt.Errorf("configured url %v is invalid", o.CouriersUrl[urlIndex])
	}
	targetUrl.Path = path.Join(targetUrl.Path, "api", "sendOrder")

	orderMessage, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("err when SendOrderMessage marshal order error: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, targetUrl.String(), bytes.NewReader(orderMessage))
	if err != nil {
		return fmt.Errorf("err when SendOrderMessage generate http request to url [%s] error: %v", targetUrl.String(), err)
	}

	res, err := o.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("err when SendOrderMessage call url [%s] error: %v", targetUrl.String(), err)
	}

	if res.StatusCode >= 300 {
		buf := new(strings.Builder)
		io.Copy(buf, res.Body)
		return fmt.Errorf("call courier api got status: %v error: %s", res.StatusCode, buf.String())
	}

	return nil
}

// return average delay value in seconds of requested orderType
// @param orderType string order type should be "match" or "fifo"
// @return float32 average delay in seconds
// @return error
func (o *OrderService) GetAverageDelayOfType(orderType string) (float32, error) {
	return o.Repo.GetAverageDelayOfOrderType(orderType)
}
