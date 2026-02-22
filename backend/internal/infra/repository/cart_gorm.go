package repository

import (
	"app/internal/domain/model"
	repo "app/internal/repository"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CartGormRepository struct {
	db *gorm.DB
}

// DI
func NewCartGormRepository(db *gorm.DB) *CartGormRepository {
	return &CartGormRepository{db: db}
}

// ユーザーのACTIVEカートを取得し、無ければ作成
func (r *CartGormRepository) GetOrCreateActiveByUserID(ctx context.Context, userID int64) (model.Cart, error) {

	var cart model.Cart

	//トランザクションで探す→無ければ作る
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		findErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND status = ?", userID, model.CartStatusActive).
			Order("id desc").
			First(&cart).Error

		if findErr == nil {
			return nil
		}

		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		// 無ければ作る
		now := time.Now()
		newCart := model.Cart{
			UserID:    userID,
			Status:    model.CartStatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := tx.Create(&newCart).Error; err != nil {
			retryErr := tx.
				Where("user_id = ? AND status = ?", userID, model.CartStatusActive).
				Order("id desc").
				First(&cart).Error
			if retryErr == nil {
				return nil
			}
			return err
		}

		cart = newCart
		return nil
	})

	if err != nil {
		return model.Cart{}, err
	}
	return cart, nil
}

// ユーザーのACTIVEカートを取得
func (r *CartGormRepository) FindActiveByUserID(ctx context.Context, userID int64) (model.Cart, error) {
	var cart model.Cart

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.CartStatusActive).
		Order("id desc").
		First(&cart).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Cart{}, repo.ErrNotFound
	}
	if err != nil {
		return model.Cart{}, err
	}
	return cart, nil
}

// carts.statusを更新
func (r *CartGormRepository) UpdateStatus(ctx context.Context, cartID int64, status model.CartStatus) error {
	res := r.db.WithContext(ctx).
		Model(&model.Cart{}).
		Where("id = ?", cartID).
		Update("status", status)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 指定カートの明細を全削除
func (r *CartGormRepository) Clear(ctx context.Context, cartID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cart model.Cart
		if err := tx.Where("id = ?", cartID).First(&cart).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrNotFound
			}
			return err
		}

		//cart_itemsを全削除
		if err := tx.Where("cart_id = ?", cartID).Delete(&model.CartItem{}).Error; err != nil {
			return err
		}

		return nil
	})
}

// カート明細を一覧取得
func (r *CartGormRepository) ListByCartID(ctx context.Context, cartID int64) ([]model.CartItem, error) {
	var items []model.CartItem

	if err := r.db.WithContext(ctx).
		Where("cart_id = ?", cartID).
		Order("id asc").
		Find(&items).Error; err != nil {
		return []model.CartItem{}, err
	}

	return items, nil
}

// 同一商品は数量加算
func (r *CartGormRepository) UpsertByCartAndProduct(ctx context.Context, cartID int64, productID int64, addQty int64, unitPriceSnapshot int64) error {

	if addQty <= 0 {
		return errors.New("invalid quantity")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.CartItem

		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("cart_id = ? AND product_id = ?", cartID, productID).
			First(&item).Error

		if err == nil {
			// 既存ありだったら数量を増やす
			newQty := item.Quantity + addQty

			res := tx.Model(&model.CartItem{}).
				Where("id = ?", item.ID).
				Update("quantity", newQty)

			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return repo.ErrNotFound
			}
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		//無い場合は新規作成
		now := time.Now()
		newItem := model.CartItem{
			CartID:            cartID,
			ProductID:         productID,
			Quantity:          addQty,
			UnitPriceSnapshot: unitPriceSnapshot,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := tx.Create(&newItem).Error; err != nil {
			return err
		}

		return nil
	})
}

// 明細の数量を更新
func (r *CartGormRepository) UpdateQuantity(ctx context.Context, cartItemID int64, qty int64) error {
	res := r.db.WithContext(ctx).
		Model(&model.CartItem{}).
		Where("id = ?", cartItemID).
		Update("quantity", qty)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 明細を削除
func (r *CartGormRepository) DeleteByID(ctx context.Context, cartItemID int64) error {
	res := r.db.WithContext(ctx).Delete(&model.CartItem{}, cartItemID)

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 明細を取得
func (r *CartGormRepository) FindByID(ctx context.Context, cartItemID int64) (model.CartItem, error) {
	var item model.CartItem

	err := r.db.WithContext(ctx).
		Where("id = ?", cartItemID).
		First(&item).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.CartItem{}, repo.ErrNotFound
	}
	if err != nil {
		return model.CartItem{}, err
	}
	return item, nil
}

//cartItemが、そのuserのカートに属しているかを判定

func (r *CartGormRepository) IsOwnedByUser(ctx context.Context, cartItemID int64, userID int64) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Table("cart_items").
		Joins("join carts on carts.id = cart_items.cart_id").
		Where("cart_items.id = ? AND carts.user_id = ?", cartItemID, userID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
