package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"app/internal/domain/model"
	coreRepo "app/internal/repository"

	"gorm.io/gorm"
)

// refreshTokenRepositoryはRefreshTokenRepositoryのGORM実装
type refreshTokenRepository struct {
	db *gorm.DB
}

// DIコンストラクター（interfaceを返すとDIが綺麗）
func NewRefreshTokenRepository(db *gorm.DB) coreRepo.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// Create はrefreshtokenを新規保存します。
func (r *refreshTokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return err
	}

	return nil
}

// FindByTokenHashはハッシュ値からrefreshtokenを1件取得します。
func (r *refreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	var t model.RefreshToken

	err := r.db.WithContext(ctx).
		First(&t, "token_hash = ?", tokenHash).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, coreRepo.ErrRefreshTokenNotFound
		}
		return nil, err
	}

	return &t, nil
}

// MarkUsedは使用済み時刻（UsedAt）を保存
func (r *refreshTokenRepository) MarkUsed(ctx context.Context, tokenID string, usedAt time.Time) error {
	tx := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ?", tokenID).
		UpdateColumn("used_at", usedAt)

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return coreRepo.ErrRefreshTokenNotFound
	}

	return nil
}

// Revokeは無効化時刻（RevokedAt）を保存
func (r *refreshTokenRepository) Revoke(ctx context.Context, tokenID string, revokedAt time.Time) error {
	tx := r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("id = ?", tokenID).
		UpdateColumn("revoked_at", revokedAt)

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return coreRepo.ErrRefreshTokenNotFound
	}

	return nil
}

// DeleteAllByUserIDはユーザーのrefreshtokenを全削除
func (r *refreshTokenRepository) DeleteAllByUserID(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.RefreshToken{}).
		Error; err != nil {
		return err
	}

	return nil
}

// DeleteByIDは特定のrefreshtokenを削除
func (r *refreshTokenRepository) DeleteByID(ctx context.Context, tokenID string) error {
	tx := r.db.WithContext(ctx).
		Where("id = ?", tokenID).
		Delete(&model.RefreshToken{})

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return coreRepo.ErrRefreshTokenNotFound
	}

	return nil
}
