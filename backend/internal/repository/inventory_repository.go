package repository

import (
	"app/internal/domain/model"
	"context"
)

type InventoryRepository interface {
	// 在庫の現在値を設定
	SetStock(ctx context.Context, productID int64, newStock int64) error

	// 在庫が足りるときだけ減算
	DecreaseStockIfEnough(ctx context.Context, productID int64, qty int64) (bool, error)

	// 在庫戻し（キャンセルなど）
	IncreaseStock(ctx context.Context, productID int64, qty int64) error

	// 調整履歴作成
	CreateAdjustment(ctx context.Context, adjustment model.InventoryAdjustment) error
}
