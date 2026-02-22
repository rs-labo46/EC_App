package model

import "time"

type RefreshToken struct {
	ID        string     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    int64      `gorm:"not null;index" json:"user_id"`
	TokenHash string     `gorm:"type:varchar(255);not null;uniqueIndex" json:"token_hash"`
	ExpiresAt time.Time  `gorm:"not null;index" json:"expires_at"`
	UsedAt    *time.Time `gorm:"index" json:"used_at"`
	UserAgent string     `gorm:"type:varchar(255);not null" json:"user_agent"`
	IP        *string    `gorm:"type:varchar(45)" json:"ip"`
	CreatedAt time.Time  `gorm:"not null" json:"created_at"`
}
