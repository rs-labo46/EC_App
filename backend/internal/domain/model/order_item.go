package model

import "time"

type OrderItem struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID             int64     `gorm:"not null;index" json:"order_id"`
	ProductID           int64     `gorm:"not null;index" json:"product_id"`
	ProductNameSnapshot string    `gorm:"type:varchar(255);not null" json:"product_name_snapshot"`
	UnitPriceSnapshot   int64     `gorm:"not null" json:"unit_price_snapshot"`
	Quantity            int64     `gorm:"not null" json:"quantity"`
	CreatedAt           time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}
