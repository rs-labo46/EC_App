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
	db *gorm.DB //DB接続（GORM）
}

// GORM実装
func NewRefreshTokenRepository(db *gorm.DB) repo.RefreshTokenRepository {
	return &refreshTokenGormRepository{db: db}
}

// リフレッシュトークンを保存し。
func (r *refreshTokenGormRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	//タイムアウトやキャンセルをDB処理に伝える
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return err
	}
	return nil
}

// token_hashで1件検索します。
func (r *refreshTokenGormRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var token model.RefreshToken

	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&token).Error

	if err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repo.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	return &token, nil
}

// used_at をセットして「使用済み」にします。
func (r *refreshTokenGormRepository) MarkUsed(ctx context.Context, tokenID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND used_at IS NULL AND revoked_at IS NULL", tokenID).
		Update("used_at", &now)

	if result.Error != nil {
		return result.Error
	}

	// 更新件数が0なら「すでに使用済み/無効/存在しない」の可能性
	if result.RowsAffected == 0 {
		return repo.ErrRefreshTokenNotFound
	}

	return nil
}

// revoked_atをセットして無効。
func (r *refreshTokenGormRepository) Revoke(ctx context.Context, tokenID string) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", tokenID).
		Update("revoked_at", &now)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return repo.ErrRefreshTokenNotFound
	}

	return nil
}

// 指定ユーザーのリフレッシュトークンを全削除します。
func (r *refreshTokenGormRepository) DeleteAllByUserID(ctx context.Context, userID string) error {
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
		return repo.ErrRefreshTokenNotFound
	}

	return nil
}
