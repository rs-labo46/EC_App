package repository

import (
	"app/internal/domain/model"
	"context"
	"errors"
	"time"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// リフレッシュトークンの保存・取得・更新・削除
type RefreshTokenRepository = interface {
	Create(ctx context.Context, token *model.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	MarkUsed(ctx context.Context, tokenID string, usedAt time.Time) error
	Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error
	DeleteAllByUserID(ctx context.Context, userID string) error
	DeleteByID(ctx context.Context, tokenID string) error
}
