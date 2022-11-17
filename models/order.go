package models

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderStatus int

const (
	OrderStarted  OrderStatus = 1
	OrderCooking  OrderStatus = 2
	OrderFinished OrderStatus = 3

	OrderIdPrefix string = "ORDER"
)

// Order model
type OrderModel struct {
	OrderId     string    `gorm:"primaryKey size:32"`
	OrderType   string    `gorm:"size:32"`
	CreatedAt   time.Time `gorm:"autoCreateTime:milli"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime:milli"`
	DeletedAt   gorm.DeletedAt
	Id          string `gorm:"uniqueIndex size:191"`
	Name        string
	PrepTime    int
	OrderStatus OrderStatus
}

// because current server is singleton,
// so we use db to create an unique primary key
// please use transaction to execute this function
// id format: {OrderIdPrefix}000000001
func (model *OrderModel) GenerateUniqueKey(db *gorm.DB) (string, error) {
	var latestModel OrderModel
	result := db.Set("gorm:query_option", "FOR UPDATE").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "order_id"}, Desc: true}).
		Last(&latestModel)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Sprintf("%s%09d", OrderIdPrefix, 1), nil
	} else if result.Error != nil {
		return "", fmt.Errorf("access db error %v", result.Error)
	}
	surfix := latestModel.OrderId[len(OrderIdPrefix):]
	index, err := strconv.Atoi(surfix)
	if err != nil {
		return "", fmt.Errorf("db dirty data: %s %v", latestModel.OrderId, err)
	}

	return fmt.Sprintf("%s%09d", OrderIdPrefix, index+1), nil
}
