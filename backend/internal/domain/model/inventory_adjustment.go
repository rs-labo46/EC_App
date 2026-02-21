package model

import "time"

//在庫調整の履歴

type InventoryAdjustment struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProductID   int64     `gorm:"not null;index" json:"product_id"`
	AdminUserID int64     `gorm:"not null;index" json:"admin_user_id"`
	Delta       int64     `gorm:"not null" json:"delta"`
	Reason      string    `gorm:"type:varchar(255);not null" json:"reason"`
	CreatedAt   time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
}
