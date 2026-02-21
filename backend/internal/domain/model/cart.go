package model

import "time"

type CartStatus string

const (
	CartStatusActive     CartStatus = "ACTIVE"
	CartStatusCheckedOut CartStatus = "CHECKED_OUT"
	CartStatusAbandoned  CartStatus = "ABANDONED"
)

// 1ユーザーにつきACTIVEは1つ
type Cart struct {
	ID        int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64      `gorm:"not null;index" json:"user_id"`
	Status    CartStatus `gorm:"type:varchar(20);not null;index" json:"status"`
	CreatedAt time.Time  `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;autoUpdateTime" json:"updated_at"`
}
