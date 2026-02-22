package repository

import "context"

// トランザクション内で使う約束
type TxRepos interface {
	Orders() OrderRepository
	OrderItems() OrderItemRepository
	Carts() CartRepository
	CartItems() CartItemRepository
	Inventory() InventoryRepository
	Products() ProductRepository
}

// UsecaseからTxの開始/commit/rollbackを隠す。
type TransactionManager interface {
	WithinTx(ctx context.Context, fn func(r TxRepos) error) error
}
