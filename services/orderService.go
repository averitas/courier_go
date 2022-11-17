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
	"github.com/averitas/courier_go/tools/logger"
	"github.com/averitas/courier_go/types"
	"gorm.io/gorm"
)

type OrderService struct {
	QueueManager tools.IQueueManager
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

// @description Get the single latest order with 'id' in order struct
// @param order *types.Order order received from api
// @return error
func (o *OrderService) GetOrderModel(order *types.Order) (res *models.OrderModel, err error) {
	err = db.Db.Where("id = ?", order.Id).Last(&res).Error
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
	err = o.saveModel(model)
	if err != nil {
		return fmt.Errorf("order set status to cooking err: %v", err)
	}
	logger.InfoLogger.Printf("Order [%s] started cooking\n", model.OrderId)

	// wait order
	time.Sleep(time.Duration(model.PrepTime) * time.Second)

	// order is done, move status to finished
	model.OrderStatus = models.OrderFinished
	err = o.saveModel(model)
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

// return average delay value in seconds of requested orderType
// @param orderType string order type should be "match" or "fifo"
// @return float32 average delay in seconds
// @return error
func (o *OrderService) GetAverageDelayOfType(orderType string) (float32, error) {
	subQuery := db.Db.Select("DATE_SUB(timediff(updated_at, created_at), INTERVAL prep_time second) AS pickup_delay").
		Where("order_type = ?", orderType).Table("order_models")
	var result float32
	err := db.Db.Select("AVG(tt.pickup_delay) as avgdelay").Table("(?) as tt", subQuery).Pluck("avgdelay", &result).Error
	return result, err
}
