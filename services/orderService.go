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

	"github.com/averitas/courier_go/db"
	"github.com/averitas/courier_go/models"
	"github.com/averitas/courier_go/tools"
	"github.com/averitas/courier_go/types"
	"gorm.io/gorm"
)

type OrderService struct {
	QueueManager tools.IQueueManager
	CouriersUrl  []string
}

func (o *OrderService) SaveOrder(order *types.Order) error {
	orderModel := &models.OrderModel{
		Id:          order.Id,
		PrepTime:    order.PrepTime,
		Name:        order.Name,
		OrderStatus: models.OrderStarted,
	}
	err := db.Db.Transaction(func(tx *gorm.DB) error {
		var erri error
		orderModel.OrderId, erri = orderModel.GenerateUniqueKey(tx)
		if erri != nil {
			return fmt.Errorf("generate id error %v", erri)
		}
		erri = tx.Create(orderModel).Error
		if erri != nil {
			return fmt.Errorf("save order error %v", erri)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("save order error: %v", err)
	}
	return nil
}

func (o *OrderService) saveModel(order *models.OrderModel) error {
	return db.Db.Save(order).Error
}

func (o *OrderService) GetOrderModel(order *types.Order) (res *models.OrderModel, err error) {
	err = db.Db.Where("id = ?", order.Id).Last(&res).Error
	return
}

func (o *OrderService) WaitUntilOrderCooked(model *models.OrderModel) (err error) {
	defer func() {
		if rcy := recover(); rcy != nil {
			err = fmt.Errorf("handler error: %v\n!panic: %v", err, rcy)
		}
	}()

	// set order status to cooking
	model.OrderStatus = models.OrderCooking
	err = o.saveModel(model)
	if err != nil {
		return fmt.Errorf("order set status to cooking err: %v", err)
	}
	fmt.Printf("Order [%s] started cooking\n", model.OrderId)

	// wait order
	time.Sleep(time.Duration(model.PrepTime) * time.Second)

	// order is done, move status to finished
	model.OrderStatus = models.OrderFinished
	err = o.saveModel(model)
	if err != nil {
		return fmt.Errorf("order set status to finished err: %v", err)
	}
	fmt.Printf("Order [%s] done!\n", model.OrderId)
	return nil
}

func (o *OrderService) SendOrderMessage(order *types.Order) error {
	err := o.QueueManager.Send(order)
	if err != nil {
		return fmt.Errorf("send message error: %v", err)
	}
	return nil
}

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

	res, err := http.DefaultClient.Do(req)
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
