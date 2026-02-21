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
	Create(ctx context.Context, product model.Product) (model.Product, error)
	FindByID(ctx context.Context, productID int64) (model.Product, error)
	// 公開商品の一覧
	ListPublic(ctx context.Context, query ProductListQuery) ([]model.Product, int64, error)
	Update(ctx context.Context, product model.Product) error
	SoftDelete(ctx context.Context, productID int64) error
	//公開切替
	SetActive(ctx context.Context, productID int64, isActive bool) error
}
