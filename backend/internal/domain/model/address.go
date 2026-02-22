package model

import "time"

// 配送先住所
type Address struct {
	ID     int64 `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID int64 `gorm:"not null;index" json:"user_id"`
	//郵便番号
	PostalCode string `gorm:"type:varchar(20);not null" json:"postal_code"`

	//都道府県
	Prefecture string `gorm:"type:varchar(100);not null" json:"prefecture"`

	//市区町村
	City string `gorm:"type:varchar(255);not null" json:"city"`

	//番地など
	Line1 string `gorm:"type:varchar(255);not null" json:"line1"`

	//建物名など
	Line2 string `gorm:"type:varchar(255)" json:"line2"`

	//宛名
	Name string `gorm:"type:varchar(255);not null" json:"name"`

	//電話番号
	Phone string `gorm:"type:varchar(30)" json:"phone"`

	//このユーザーのデフォルト住所か
	IsDefault bool `gorm:"not null;default:false" json:"is_default"`

	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}
