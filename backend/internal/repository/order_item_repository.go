package repository

import (
	"context"

	"app/internal/domain/model"
)

type OrderItemRepository interface {
	CreateBulk(ctx context.Context, orderID int64, items []model.OrderItem) error
	ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderItem, error)
}
