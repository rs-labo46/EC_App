package repository

import (
	"context"
	"errors"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type InventoryGormRepository struct {
	db *gorm.DB
}

func NewInventoryGormRepository(db *gorm.DB) *InventoryGormRepository {
	return &InventoryGormRepository{db: db}
}

// 在庫の現在値を設定
func (r *InventoryGormRepository) SetStock(ctx context.Context, productID int64, newStock int64) error {
	res := r.db.WithContext(ctx).
		Model(&model.Product{}).
		Where("id = ?", productID).
		Update("stock", newStock)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 在庫が足りるときだけ減らす
func (r *InventoryGormRepository) DecreaseStockIfEnough(ctx context.Context, productID int64, qty int64) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&model.Product{}).
		Where("id = ? AND stock >= ?", productID, qty).
		Update("stock", gorm.Expr("stock - ?", qty))

	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}
	return true, nil
}

// 在庫戻し（キャンセル）
func (r *InventoryGormRepository) IncreaseStock(ctx context.Context, productID int64, qty int64) error {
	res := r.db.WithContext(ctx).
		Model(&model.Product{}).
		Where("id = ?", productID).
		Update("stock", gorm.Expr("stock + ?", qty))

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 調整履歴作成
func (r *InventoryGormRepository) CreateAdjustment(ctx context.Context, adj model.InventoryAdjustment) error {
	if err := r.db.WithContext(ctx).Create(&adj).Error; err != nil {
		return err
	}
	return nil
}

func isNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
