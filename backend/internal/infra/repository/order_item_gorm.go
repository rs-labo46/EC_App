package repository

import (
	"context"
	"errors"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type OrderItemGormRepository struct {
	db *gorm.DB
}

func NewOrderItemGormRepository(db *gorm.DB) *OrderItemGormRepository {
	return &OrderItemGormRepository{db: db}
}

func (r *OrderItemGormRepository) CreateBulk(ctx context.Context, orderID int64, items []model.OrderItem) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		items[i].OrderID = orderID
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return err
	}
	return nil
}

func (r *OrderItemGormRepository) ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderItem, error) {
	var items []model.OrderItem
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Order("id asc").Find(&items).Error
	if err != nil {
		return []model.OrderItem{}, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []model.OrderItem{}, repo.ErrNotFound
	}
	return items, nil
}
