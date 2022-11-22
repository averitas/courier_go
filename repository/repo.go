package repository

import (
	"fmt"

	"github.com/averitas/courier_go/db"
	"github.com/averitas/courier_go/models"
	"gorm.io/gorm"
)

type IOrderRepo interface {
	CreateOrder(*models.OrderModel) error

	// Upsert order model into database
	SaveModel(*models.OrderModel) error

	// Get order by @field OrderModel.Id
	GetOrderById(string) (*models.OrderModel, error)
	// Calculate average delay of order type filtered by
	// @field: OrderModel.OrderType
	GetAverageDelayOfOrderType(string) (float32, error)
}

type OrderRepo struct {
}

func (r *OrderRepo) SaveModel(order *models.OrderModel) error {
	return db.Db.Save(order).Error
}

func (r *OrderRepo) GetOrderById(id string) (res *models.OrderModel, err error) {
	err = db.Db.Where("id = ?", id).Last(&res).Error
	return
}

func (r *OrderRepo) GetAverageDelayOfOrderType(orderType string) (float32, error) {
	subQuery := db.Db.Select("DATE_SUB(timediff(updated_at, created_at), INTERVAL prep_time second) AS pickup_delay").
		Where("order_type = ?", orderType).Table("order_models")
	var result float32
	err := db.Db.Select("AVG(tt.pickup_delay) as avgdelay").Table("(?) as tt", subQuery).Pluck("avgdelay", &result).Error
	return result, err
}

func (r *OrderRepo) CreateOrder(orderModel *models.OrderModel) error {
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

	return err
}
