package repository

import (
	"context"
	"errors"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type OrderGormRepository struct {
	db *gorm.DB
}

func NewOrderGormRepository(db *gorm.DB) *OrderGormRepository {
	return &OrderGormRepository{db: db}
}

func (r *OrderGormRepository) FindByID(ctx context.Context, orderID int64) (model.Order, error) {
	var o model.Order
	err := r.db.WithContext(ctx).Where("id = ?", orderID).First(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Order{}, repo.ErrNotFound
	}
	if err != nil {
		return model.Order{}, err
	}
	return o, nil
}

func (r *OrderGormRepository) ListByUserID(ctx context.Context, userID int64, page int, limit int) ([]model.Order, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return []model.Order{}, 0, err
	}

	var items []model.Order
	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return []model.Order{}, 0, err
	}

	return items, total, nil
}

func (r *OrderGormRepository) Create(ctx context.Context, order model.Order) (int64, error) {
	if err := r.db.WithContext(ctx).Create(&order).Error; err != nil {
		return 0, err
	}
	return order.ID, nil
}

func (r *OrderGormRepository) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error {
	res := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("id = ?", orderID).
		Update("status", status)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *OrderGormRepository) FindByIdempotencyKey(ctx context.Context, userID int64, key string) (model.Order, bool, error) {
	var o model.Order
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND idempotency_key = ?", userID, key).
		First(&o).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Order{}, false, nil
	}
	if err != nil {
		return model.Order{}, false, err
	}
	return o, true, nil
}

func (r *OrderGormRepository) ListAdmin(ctx context.Context, f repo.AdminOrderListFilter) ([]model.Order, int64, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 50
	}

	q := r.db.WithContext(ctx).Model(&model.Order{})

	//status 絞り込み
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	//user_id 絞り込み
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}

	//期間絞り込み
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return []model.Order{}, 0, err
	}

	var items []model.Order
	offset := (f.Page - 1) * f.Limit
	if err := q.Order("id desc").Limit(f.Limit).Offset(offset).Find(&items).Error; err != nil {
		return []model.Order{}, 0, err
	}

	return items, total, nil
}
