package repository

import (
	"context"
	"errors"
	"time"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type refreshTokenGormRepository struct {
	db *gorm.DB
}

// DI
func NewRefreshTokenGormRepository(db *gorm.DB) repo.RefreshTokenRepository {
	return &refreshTokenGormRepository{db: db}
}

// リフレッシュトークンを保存する
func (r *refreshTokenGormRepository) Create(ctx context.Context, token model.RefreshToken) error {
	if err := r.db.WithContext(ctx).Create(&token).Error; err != nil {
		return err
	}
	return nil
}

// token_hash で1件検索。
func (r *refreshTokenGormRepository) FindByHash(ctx context.Context, tokenHash string) (model.RefreshToken, bool, error) {
	var token model.RefreshToken

	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&token).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RefreshToken{}, false, nil
		}
		return model.RefreshToken{}, false, err
	}

	return token, true, nil
}

// used_at をセットして使用済み
func (r *refreshTokenGormRepository) MarkUsed(ctx context.Context, tokenID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND used_at IS NULL", tokenID).
		Update("used_at", &now)

	if result.Error != nil {
		return result.Error
	}

	//更新件数が0なら「存在しない or すでに使用済み」
	if result.RowsAffected == 0 {
		return errors.New("refresh token not found or already used")
	}

	return nil
}

// 指定ユーザーのリフレッシュトークンを全削除。
func (r *refreshTokenGormRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.RefreshToken{}).Error; err != nil {
		return err
	}
	return nil
}

// 指定IDのリフレッシュトークンを削除。
func (r *refreshTokenGormRepository) DeleteByID(ctx context.Context, tokenID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", tokenID).
		Delete(&model.RefreshToken{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("refresh token not found")
	}

	return nil
}

// 期限切れの refresh を削除し、削除件数を返す。
func (r *refreshTokenGormRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", now).
		Delete(&model.RefreshToken{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}
