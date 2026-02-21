package repository

import (
	"app/internal/domain/model"
	"context"
	"errors"
)

var ErrNotFound = errors.New("not found")

// 一覧検索
type ProductListQuery struct {
	Page     int
	Limit    int
	Q        string
	MinPrice *int64
	MaxPrice *int64
	Sort     string
}

// 商品の永続化（保存・取得）だけを約束。
type ProductRepository interface {
	ListPublic(ctx context.Context, q ProductListQuery) ([]model.Product, int64, error)
	FindByID(ctx context.Context, id int64) (model.Product, error)

	Create(ctx context.Context, p model.Product) (model.Product, error)
	Update(ctx context.Context, p model.Product) error
	SoftDelete(ctx context.Context, id int64) error
}

// 在庫の永続化と履歴保存をまとめた約束。
type InventoryRepository interface {
	SetStockWithAdjustment(ctx context.Context, adminUserID int64, productID int64, newStock int64, reason string) error
}
