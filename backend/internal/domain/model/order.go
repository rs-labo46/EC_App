package model

import "time"

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusPaid     OrderStatus = "PAID"
	OrderStatusShipped  OrderStatus = "SHIPPED"
	OrderStatusCanceled OrderStatus = "CANCELED"
)

type Order struct {
	ID             int64       `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         int64       `gorm:"not null;index" json:"user_id"`
	AddressID      int64       `gorm:"not null" json:"address_id"`
	Status         OrderStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	TotalPrice     int64       `gorm:"not null" json:"total_price"`
	IdempotencyKey string      `gorm:"type:varchar(255);not null;uniqueIndex" json:"-"`
	CreatedAt      time.Time   `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time   `gorm:"not null;autoUpdateTime" json:"updated_at"`
}
