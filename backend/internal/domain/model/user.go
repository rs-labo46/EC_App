package model

import "time"

// 作成・更新日時
type Timestamp struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ユーザー権限
type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

// Userのエンティティ
type User struct {
	ID           string     `json:"id" gorm:"type:uuid;primaryKey"`
	Email        string     `json:"email" gorm:"uniqueIndex;not null"`
	Password     string     `json:"password" gorm:"not null"`
	Role         Role       `json:"role" gorm:"type:varchar(20);not null;default:'USER'"`
	TokenVersion int        `json:"tokenVersion" gorm:"not null;default:0"`
	IsActive     bool       `json:"isActive" gorm:"not null;default:true"`
	LastLoginAt  *time.Time `json:"lastLoginAt" gorm:""`

	Timestamps Timestamp `json:"timestamps" gorm:"embedded"`
}
