package repository

import (
	"context"
	"errors"
	"strings"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type ProductGormRepository struct {
	db *gorm.DB
}

// DI
func NewProductGormRepository(db *gorm.DB) *ProductGormRepository {
	return &ProductGormRepository{db: db}
}

// 公開商品のみを、検索/価格帯/ソート/ページング付きで返す。
func (r *ProductGormRepository) ListPublic(ctx context.Context, q repo.ProductListQuery) ([]model.Product, int64, error) {
	var products []model.Product
	var total int64

	tx := r.db.WithContext(ctx).Model(&model.Product{})

	// 公開（is_active=true）かつ、商品削除されていないものだけ
	tx = tx.Where("is_active = ?", true)

	// q nameを対象
	if strings.TrimSpace(q.Q) != "" {
		like := "%" + strings.TrimSpace(q.Q) + "%"
		tx = tx.Where("name ILIKE ?", like)
	}

	//価格帯
	if q.MinPrice != nil {
		tx = tx.Where("price >= ?", *q.MinPrice)
	}
	if q.MaxPrice != nil {
		tx = tx.Where("price <= ?", *q.MaxPrice)
	}

	//total（件数）
	if err := tx.Count(&total).Error; err != nil {
		return []model.Product{}, 0, err
	}

	//sort
	switch q.Sort {
	case "price_asc":
		tx = tx.Order("price asc").Order("id asc")
	case "price_desc":
		tx = tx.Order("price desc").Order("id desc")
	default:
		tx = tx.Order("created_at desc").Order("id desc")
	}

	offset := (q.Page - 1) * q.Limit
	if err := tx.Offset(offset).Limit(q.Limit).Find(&products).Error; err != nil {
		return []model.Product{}, 0, err
	}

	return products, total, nil
}

// IDで商品を取得
func (r *ProductGormRepository) FindByID(ctx context.Context, id int64) (model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Product{}, repo.ErrNotFound
	}
	if err != nil {
		return model.Product{}, err
	}
	return p, nil
}

// 商品の作成
func (r *ProductGormRepository) Create(ctx context.Context, p model.Product) (model.Product, error) {
	if err := r.db.WithContext(ctx).Create(&p).Error; err != nil {
		return model.Product{}, err
	}
	return p, nil
}

// 商品の更新
func (r *ProductGormRepository) Update(ctx context.Context, p model.Product) error {
	res := r.db.WithContext(ctx).Model(&model.Product{}).Where("id = ?", p.ID).Updates(map[string]interface{}{
		"name":        p.Name,
		"description": p.Description,
		"price":       p.Price,
		"stock":       p.Stock,
		"is_active":   p.IsActive,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 商品削除
func (r *ProductGormRepository) SoftDelete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Product{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// 在庫を「現在値」に更新し、調整履歴も残す
func (r *ProductGormRepository) SetStockWithAdjustment(ctx context.Context, adminUserID int64, productID int64, newStock int64, reason string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//現在の在庫を取得
		var p model.Product
		if err := tx.First(&p, productID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repo.ErrNotFound
			}
			return err
		}

		//products.stockを更新
		res := tx.Model(&model.Product{}).
			Where("id = ?", productID).
			Update("stock", newStock)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return repo.ErrNotFound
		}

		//adjustmentsを作成
		adj := model.InventoryAdjustment{
			ProductID:   productID,
			AdminUserID: adminUserID,
			Delta:       newStock - p.Stock,
			Reason:      reason,
		}
		if err := tx.Create(&adj).Error; err != nil {
			return err
		}

		return nil
	})
}
