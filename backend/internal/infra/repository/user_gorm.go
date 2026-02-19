package repository

import (
	"context"
	"errors"
	"fmt"

	"app/internal/domain/model"
	coreRepo "app/internal/repository"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB // DB接続
}

// DIコンストラクター
func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

// 新規ユーザー作成
func (ur *userRepository) Create(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	// context をGORMへ渡す
	if err := ur.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}

	return nil
}

// IDからユーザー取得
func (ur *userRepository) FindByID(ctx context.Context, userID string) (*model.User, error) {
	var u model.User

	err := ur.db.WithContext(ctx).
		First(&u, "id = ?", userID).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, coreRepo.ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// emailからユーザー取得（loginで使う）
func (ur *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User

	err := ur.db.WithContext(ctx).
		First(&u, "email = ?", email).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, coreRepo.ErrUserNotFound
		}
		return nil, err
	}

	return &u, nil
}

// ユーザー情報の更新
func (ur *userRepository) Update(ctx context.Context, user *model.User) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	err := ur.db.WithContext(ctx).Save(user).Error
	if err != nil {
		return err
	}

	return nil
}

// token_versionをDB上で+1
func (ur *userRepository) IncrementTokenVersion(ctx context.Context, userID string) error {
	tx := ur.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + ?", 1))

	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return coreRepo.ErrUserNotFound
	}

	return nil
}
