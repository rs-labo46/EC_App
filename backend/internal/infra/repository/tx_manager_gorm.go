package repository

import (
	"context"

	repo "app/internal/repository"

	"gorm.io/gorm"
)

type txReposGorm struct {
	orders     repo.OrderRepository
	orderItems repo.OrderItemRepository
	carts      repo.CartRepository
	cartItems  repo.CartItemRepository
	inventory  repo.InventoryRepository
	products   repo.ProductRepository
}

func (r *txReposGorm) Orders() repo.OrderRepository         { return r.orders }
func (r *txReposGorm) OrderItems() repo.OrderItemRepository { return r.orderItems }
func (r *txReposGorm) Carts() repo.CartRepository           { return r.carts }
func (r *txReposGorm) CartItems() repo.CartItemRepository   { return r.cartItems }
func (r *txReposGorm) Inventory() repo.InventoryRepository  { return r.inventory }
func (r *txReposGorm) Products() repo.ProductRepository     { return r.products }

type TxManagerGorm struct {
	db *gorm.DB
}

func NewTxManagerGorm(db *gorm.DB) *TxManagerGorm {
	return &TxManagerGorm{db: db}
}

func (tm *TxManagerGorm) WithinTx(ctx context.Context, fn func(r repo.TxRepos) error) error {
	return tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//repoはtxを持ったDBで作り直す
		r := &txReposGorm{
			orders:     NewOrderGormRepository(tx),
			orderItems: NewOrderItemGormRepository(tx),
			carts:      NewCartGormRepository(tx),
			cartItems:  NewCartGormRepository(tx),
			inventory:  NewInventoryGormRepository(tx),
			products:   NewProductGormRepository(tx),
		}
		return fn(r)
	})
}
