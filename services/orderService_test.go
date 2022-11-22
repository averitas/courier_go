package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/averitas/courier_go/mocks"
	"github.com/averitas/courier_go/models"
	"github.com/averitas/courier_go/types"
	"github.com/golang/mock/gomock"
)

var (
	mockCtrl    *gomock.Controller
	testService *OrderService

	mockRepo         *mocks.MockIOrderRepo
	mockHttpClient   *mocks.MockHttpClient
	mockQueueManager *mocks.MockIQueueManager
)

func TestMain(m *testing.M) {
	fmt.Println("Prepare tests")

	testService = &OrderService{
		Repo:         nil,
		HttpClient:   nil,
		QueueManager: nil,
		CouriersUrl:  nil,
	}
	fmt.Println("Test started")

	var retval = m.Run()

	fmt.Println("Test ended")
	os.Exit(retval)
}

func TestCreateOrder(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	setup()

	// mock structs
	order := &types.Order{
		Id:       "id123",
		Name:     "n123",
		PrepTime: 1,
	}
	// orderModel := &models.OrderModel{
	// 	Id:       order.Id,
	// 	Name:     order.Name,
	// 	PrepTime: order.PrepTime,
	// }

	// set mock controller
	gomock.InOrder(
		mockRepo.EXPECT().CreateOrder(gomock.Any()).Return(nil),
	)

	// begin test
	testService.SaveOrder(order)

	// Finished
	tearDown()
}

func TestCallRandomCourierAPI(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	setup()

	// mock structs
	order := &types.Order{
		Id:       "id123",
		Name:     "n123",
		PrepTime: 1,
	}
	orderMessage, err := json.Marshal(order)
	if err != nil {
		t.Errorf("err when SendOrderMessage marshal order error: %v", err)
	}

	// set mock controller
	gomock.InOrder(
		mockHttpClient.EXPECT().Do(gomock.Any()).DoAndReturn(
			func(req *http.Request) (*http.Response, error) {
				if !strings.HasPrefix(req.URL.String(), "http://test.com/api/sendOrder") {
					return nil, fmt.Errorf("invalid url")
				}

				buf := new(strings.Builder)
				io.Copy(buf, req.Body)
				if buf.String() != string(orderMessage) {
					return &http.Response{StatusCode: http.StatusBadRequest}, nil
				}

				return &http.Response{StatusCode: http.StatusOK}, nil
			},
		),
	)

	// begin test
	err = testService.CallRandomCourierAPI(order)
	if err != nil {
		t.Error(err)
	}

	// Finished
	tearDown()
}

func TestWaitUntilOrderCooked(t *testing.T) {
	mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	setup()

	// mock structs
	order := &types.Order{
		Id:       "id123",
		Name:     "n123",
		PrepTime: 1,
	}
	orderModel := &models.OrderModel{
		OrderId:     "testid",
		OrderType:   "fifo",
		OrderStatus: models.OrderStarted,
		Id:          order.Id,
		Name:        order.Name,
		PrepTime:    order.PrepTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// set mock
	mockRepo.EXPECT().GetOrderById(gomock.Eq(order.Id)).Return(
		&models.OrderModel{
			OrderId:     "testid",
			OrderType:   "fifo",
			OrderStatus: models.OrderStarted,
			Id:          order.Id,
			Name:        order.Name,
			PrepTime:    order.PrepTime,
			CreatedAt:   orderModel.CreatedAt,
			UpdatedAt:   orderModel.UpdatedAt,
		}, nil)
	gomock.InOrder(
		mockRepo.EXPECT().SaveModel(gomock.Any()).DoAndReturn(
			func(m *models.OrderModel) error {
				if m.OrderStatus != models.OrderCooking {
					return fmt.Errorf("Didn't start cooking status")
				}

				orderModel.UpdatedAt = time.Now()
				orderModel.OrderStatus = m.OrderStatus

				return nil
			},
		),
		mockRepo.EXPECT().SaveModel(gomock.Any()).DoAndReturn(
			func(m *models.OrderModel) error {
				if m.OrderStatus != models.OrderFinished {
					return fmt.Errorf("Didn't set finished status")
				}

				orderModel.UpdatedAt = time.Now()
				orderModel.OrderStatus = m.OrderStatus

				return nil
			},
		),
	)

	// begin test
	waitchannel := make(chan interface{})

	go func() {
		model1, _ := testService.GetOrderModel(order)

		var err = testService.WaitUntilOrderCooked(model1)
		if err != nil {
			t.Error(err)
		}

		waitchannel <- nil
	}()

	<-waitchannel
	if orderModel.UpdatedAt.Sub(orderModel.CreatedAt) < time.Second*time.Duration(orderModel.PrepTime) {
		t.Errorf("wait time is incorrect, waited only: time[%v]", orderModel.UpdatedAt.Sub(orderModel.CreatedAt))
	}

	// Finished
	tearDown()
}

func setup() {
	mockRepo = mocks.NewMockIOrderRepo(mockCtrl)
	mockHttpClient = mocks.NewMockHttpClient(mockCtrl)
	mockQueueManager = mocks.NewMockIQueueManager(mockCtrl)
	testService = &OrderService{
		Repo:         mockRepo,
		HttpClient:   mockHttpClient,
		QueueManager: mockQueueManager,
		CouriersUrl:  []string{"http://test.com"},
	}
}

func tearDown() {
	testService = nil
	mockCtrl = nil
	mockHttpClient = nil
	mockQueueManager = nil
	mockRepo = nil
}
