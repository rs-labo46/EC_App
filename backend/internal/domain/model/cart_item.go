package model

import "time"

// カートの明細
// 追加時点の価格）を必ず保存。
type CartItem struct {
	ID                int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CartID            int64     `gorm:"not null;index" json:"cart_id"`
	ProductID         int64     `gorm:"not null;index" json:"product_id"`
	Quantity          int64     `gorm:"not null" json:"quantity"`
	UnitPriceSnapshot int64     `gorm:"not null;column:unit_price_snapshot" json:"unit_price_snapshot"`
	CreatedAt         time.Time `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"not null;autoUpdateTime" json:"updated_at"`
}
