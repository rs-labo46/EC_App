package repository

import (
	"app/internal/domain/model"
	domainrepo "app/internal/repository"
	"context"
	"errors"

	"gorm.io/gorm"
)

type userGormRepository struct {
	db *gorm.DB
}

// DI
// main.goでこれをnewしてusecaseに注入します。
func NewUserGormRepository(db *gorm.DB) domainrepo.UserRepository {
	return &userGormRepository{db: db}
}

// Create はユーザーを新規作成
func (r *userGormRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

// emailでユーザーを1件取得
func (r *userGormRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&u).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

// IDでユーザーを1件取得
func (r *userGormRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User

	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&u).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

// ユーザーを更新。
func (r *userGormRepository) Update(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return err
	}
	return nil
}

// token_versionを+1 します。
func (r *userGormRepository) IncrementTokenVersion(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("token_version", gorm.Expr("token_version + ?", 1))

	if res.Error != nil {
		return res.Error
	}

	// 0件更新は「対象がない」
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
