package repository

import (
	"context"

	"app/internal/domain/model"
)

type CartItemRepository interface {
	ListByCartID(ctx context.Context, cartID int64) ([]model.CartItem, error)
	// 同一商品はプラス
	UpsertByCartAndProduct(ctx context.Context, cartID int64, productID int64, addQty int64, unitPriceSnapshot int64) error
	UpdateQuantity(ctx context.Context, cartItemID int64, qty int64) error
	DeleteByID(ctx context.Context, cartItemID int64) error
	FindByID(ctx context.Context, cartItemID int64) (model.CartItem, error)
	IsOwnedByUser(ctx context.Context, cartItemID int64, userID int64) (bool, error)
}
