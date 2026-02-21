package model

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Price       int64          `gorm:"not null" json:"price"`
	Stock       int64          `gorm:"not null" json:"stock"`
	IsActive    bool           `gorm:"not null;default:false" json:"is_active"`
	CreatedAt   time.Time      `gorm:"not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
