package repository

import (
	"app/internal/domain/model"
	repo "app/internal/repository"
	"context"

	"gorm.io/gorm"
)

type addressGormRepository struct {
	db *gorm.DB
}

// DI
func NewAddressGormRepository(db *gorm.DB) repo.AddressRepository {
	return &addressGormRepository{db: db}
}

// 住所を作成
func (r *addressGormRepository) Create(ctx context.Context, address model.Address) (model.Address, error) {
	if err := r.db.WithContext(ctx).Create(&address).Error; err != nil {
		return model.Address{}, err
	}
	return address, nil
}

// ユーザーの住所一覧を返す
func (r *addressGormRepository) ListByUserID(ctx context.Context, userID int64) ([]model.Address, error) {
	var list []model.Address
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, id ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// 住所IDで1件取得
func (r *addressGormRepository) FindByID(ctx context.Context, addressID int64) (model.Address, error) {
	var a model.Address
	if err := r.db.WithContext(ctx).First(&a, addressID).Error; err != nil {
		return model.Address{}, err
	}
	return a, nil
}

// 住所を更新
func (r *addressGormRepository) Update(ctx context.Context, address model.Address) error {
	result := r.db.WithContext(ctx).
		Model(&model.Address{}).
		Where("id = ?", address.ID).
		Select(
			"postal_code",
			"prefecture",
			"city",
			"line1",
			"line2",
			"name",
			"phone",
		).
		Updates(address)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// 住所を削除
func (r *addressGormRepository) Delete(ctx context.Context, addressID int64) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", addressID).
		Delete(&model.Address{})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// その住所がそのユーザーのものか
func (r *addressGormRepository) IsOwnedByUser(ctx context.Context, addressID, userID int64) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.Address{}).
		Where("id = ? AND user_id = ?", addressID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count == 1, nil
}

// デフォルト住所を切り替える
func (r *addressGormRepository) SetDefault(ctx context.Context, userID, addressID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//指定住所がこのユーザーのものか確認
		var count int64
		if err := tx.Model(&model.Address{}).
			Where("id = ? AND user_id = ?", addressID, userID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}

		//そのユーザーのdefaultを全て false
		if err := tx.Model(&model.Address{}).
			Where("user_id = ? AND is_default = TRUE", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		//指定住所だけ true
		result := tx.Model(&model.Address{}).
			Where("id = ? AND user_id = ?", addressID, userID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
