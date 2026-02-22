package repository

import (
	"context"

	"app/internal/domain/model"
)

type CartRepository interface {
	GetOrCreateActiveByUserID(ctx context.Context, userID int64) (model.Cart, error)
	FindActiveByUserID(ctx context.Context, userID int64) (model.Cart, error)
	UpdateStatus(ctx context.Context, cartID int64, status model.CartStatus) error
	Clear(ctx context.Context, cartID int64) error
}
