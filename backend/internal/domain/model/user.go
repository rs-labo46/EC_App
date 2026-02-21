package model

import "time"

type Role string

const (
	RoleUser  Role = "USER"
	RoleAdmin Role = "ADMIN"
)

type User struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"column:password_hash;not null"`
	Role         Role   `gorm:"type:varchar(20);not null;default:'USER'"`
	TokenVersion int    `gorm:"not null;default:0"`
	IsActive     bool   `gorm:"not null;default:true"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
