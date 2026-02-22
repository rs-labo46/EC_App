package repository

import (
	"context"
	"time"

	"app/internal/domain/model"
)

type AdminOrderListFilter struct {
	Page   int
	Limit  int
	Status string
	UserID *int64
	From   *time.Time
	To     *time.Time
}

type OrderRepository interface {
	FindByID(ctx context.Context, orderID int64) (model.Order, error)
	ListByUserID(ctx context.Context, userID int64, page int, limit int) ([]model.Order, int64, error)
	Create(ctx context.Context, order model.Order) (int64, error)
	UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error

	//検索（同じキーなら同じ結果を返す）
	FindByIdempotencyKey(ctx context.Context, userID int64, key string) (model.Order, bool, error)
	//管理者用の注文一覧
	ListAdmin(ctx context.Context, f AdminOrderListFilter) ([]model.Order, int64, error)
}
