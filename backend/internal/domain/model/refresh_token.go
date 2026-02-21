package model

import "time"

type RefreshToken struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	UserID    int64      `gorm:"not null;index"`
	TokenHash string     `gorm:"not null;uniqueIndex"`
	UserAgent string     `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	UsedAt    *time.Time `gorm:"index"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
