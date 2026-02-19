package model

import "time"

type RefreshToken struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    string     `json:"userId" gorm:"type:uuid;not null;index"`
	TokenHash string     `json:"-" gorm:"not null;uniqueIndex"`
	UserAgent string     `json:"userAgent" gorm:"not null"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"not null;index"`
	UsedAt    *time.Time `json:"usedAt" gorm:"index"`
	RevokedAt *time.Time `json:"revokedAt" gorm:"index"`
	Timestamp Timestamp  `json:"timestamps" gorm:"embedded"`
}
