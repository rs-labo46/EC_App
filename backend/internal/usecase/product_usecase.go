package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"app/internal/domain/model"
	repo "app/internal/repository"
)

type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

func NewHTTPError(status int, message string) error {
	return &HTTPError{
		Status:  status,
		Message: message,
	}
}

func AsHTTPError(err error) (*HTTPError, bool) {
	var he *HTTPError
	ok := errors.As(err, &he)
	return he, ok
}

type ProductUsecase struct {
	productRepo   repo.ProductRepository
	inventoryRepo repo.InventoryRepository
	auditRepo     repo.AuditLogRepository
}

// DI
func NewProductUsecase(
	productRepo repo.ProductRepository,
	inventoryRepo repo.InventoryRepository,
	auditRepo repo.AuditLogRepository, // ★追加
) *ProductUsecase {
	return &ProductUsecase{
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		auditRepo:     auditRepo,
	}
}

// GET /productsの入力DTO
type ListProductsInput struct {
	Page     int
	Limit    int
	Q        string
	MinPrice *int64
	MaxPrice *int64
	Sort     string
}

type ProductListOutput struct {
	Items []model.Product `json:"items"`
	Total int64           `json:"total"`
	Page  int             `json:"page"`
	Limit int             `json:"limit"`
}

func (u *ProductUsecase) ListPublicProducts(ctx context.Context, in ListProductsInput) (ProductListOutput, error) {
	if in.Page < 1 {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "invalid page")
	}
	if in.Limit < 1 || in.Limit > 100 {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "invalid limit")
	}
	if len(in.Q) > 100 {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "q too long")
	}
	if in.MinPrice != nil && *in.MinPrice < 0 {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "min_price must be >= 0")
	}
	if in.MaxPrice != nil && *in.MaxPrice < 0 {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "max_price must be >= 0")
	}
	if in.MinPrice != nil && in.MaxPrice != nil && *in.MinPrice > *in.MaxPrice {
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "min_price must be <= max_price")
	}
	switch in.Sort {
	case "", "new", "price_asc", "price_desc":
	default:
		return ProductListOutput{}, NewHTTPError(http.StatusBadRequest, "invalid sort")
	}

	items, total, err := u.productRepo.ListPublic(ctx, repo.ProductListQuery{
		Page:     in.Page,
		Limit:    in.Limit,
		Q:        strings.TrimSpace(in.Q),
		MinPrice: in.MinPrice,
		MaxPrice: in.MaxPrice,
		Sort:     in.Sort,
	})
	if err != nil {
		return ProductListOutput{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	return ProductListOutput{
		Items: items,
		Total: total,
		Page:  in.Page,
		Limit: in.Limit,
	}, nil
}

func (u *ProductUsecase) GetProductDetail(ctx context.Context, productID int64) (model.Product, error) {
	if productID <= 0 {
		return model.Product{}, NewHTTPError(http.StatusBadRequest, "invalid product id")
	}

	p, err := u.productRepo.FindByID(ctx, productID)
	if err == repo.ErrNotFound {
		return model.Product{}, NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return model.Product{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	if !p.IsActive {
		return model.Product{}, NewHTTPError(http.StatusNotFound, "not found")
	}
	return p, nil
}

type AdminCreateProductInput struct {
	Name        string
	Description string
	Price       int64
	Stock       int64
	IsActive    bool
}

func (u *ProductUsecase) AdminCreateProduct(ctx context.Context, adminUserID int64, in AdminCreateProductInput) (int64, error) {
	if adminUserID <= 0 {
		return 0, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if strings.TrimSpace(in.Name) == "" {
		return 0, NewHTTPError(http.StatusBadRequest, "name required")
	}
	if in.Price < 0 {
		return 0, NewHTTPError(http.StatusBadRequest, "price must be >= 0")
	}
	if in.Stock < 0 {
		return 0, NewHTTPError(http.StatusBadRequest, "stock must be >= 0")
	}

	now := time.Now()
	p, err := u.productRepo.Create(ctx, model.Product{
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
		IsActive:    in.IsActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return 0, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	return p.ID, nil
}

func (u *ProductUsecase) AdminUpdateProduct(ctx context.Context, adminUserID int64, productID int64, in AdminCreateProductInput) error {
	if adminUserID <= 0 {
		return NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if productID <= 0 {
		return NewHTTPError(http.StatusBadRequest, "invalid product id")
	}
	if strings.TrimSpace(in.Name) == "" {
		return NewHTTPError(http.StatusBadRequest, "name required")
	}
	if in.Price < 0 {
		return NewHTTPError(http.StatusBadRequest, "price must be >= 0")
	}
	if in.Stock < 0 {
		return NewHTTPError(http.StatusBadRequest, "stock must be >= 0")
	}

	err := u.productRepo.Update(ctx, model.Product{
		ID:          productID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Price:       in.Price,
		Stock:       in.Stock,
		IsActive:    in.IsActive,
		UpdatedAt:   time.Now(),
	})
	if err == repo.ErrNotFound {
		return NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}
	return nil
}

func (u *ProductUsecase) AdminDeleteProduct(ctx context.Context, adminUserID int64, productID int64) error {
	if adminUserID <= 0 {
		return NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if productID <= 0 {
		return NewHTTPError(http.StatusBadRequest, "invalid product id")
	}

	err := u.productRepo.SoftDelete(ctx, productID)
	if err == repo.ErrNotFound {
		return NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}
	return nil
}

func (u *ProductUsecase) AdminUpdateInventory(ctx context.Context, adminUserID int64, productID int64, newStock int64, reason string) error {
	if adminUserID <= 0 {
		return NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if productID <= 0 {
		return NewHTTPError(http.StatusBadRequest, "invalid product id")
	}
	if newStock < 0 {
		return NewHTTPError(http.StatusBadRequest, "stock must be >= 0")
	}
	if strings.TrimSpace(reason) == "" {
		return NewHTTPError(http.StatusBadRequest, "reason required")
	}

	//変更前の在庫（before）
	p, err := u.productRepo.FindByID(ctx, productID)
	if err == repo.ErrNotFound {
		return NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}

	beforeJSON := fmt.Sprintf(`{"stock":%d}`, p.Stock)
	afterJSON := fmt.Sprintf(`{"stock":%d}`, newStock)

	//在庫の現在値を更新
	if err := u.inventoryRepo.SetStock(ctx, productID, newStock); err != nil {
		if err == repo.ErrNotFound {
			return NewHTTPError(http.StatusNotFound, "not found")
		}
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}

	//履歴を作成（差分）
	adj := model.InventoryAdjustment{
		ProductID:   productID,
		AdminUserID: adminUserID,
		Delta:       newStock - p.Stock,
		Reason:      strings.TrimSpace(reason),
		CreatedAt:   time.Now(),
	}
	if err := u.inventoryRepo.CreateAdjustment(ctx, adj); err != nil {
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}

	//監査ログを作成（在庫更新）
	//「誰が」「何を」「どの対象に」「どう変えたか」を残す
	if err := u.auditRepo.Create(ctx, model.AuditLog{
		ActorUserID:  adminUserID,
		Action:       model.AuditActionUpdateStock,
		ResourceType: model.AuditResourceProduct,
		ResourceID:   productID,
		BeforeJSON:   beforeJSON,
		AfterJSON:    afterJSON,
		CreatedAt:    time.Now(),
	}); err != nil {
		return NewHTTPError(http.StatusInternalServerError, "db error")
	}

	return nil
}
