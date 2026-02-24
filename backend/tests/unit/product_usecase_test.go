package unit

import (
	"app/internal/domain/model"
	repo "app/internal/repository"
	"app/internal/usecase"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =====================
// Mocks（衝突回避の命名）
// =====================

type ProdProductRepoMock struct{ mock.Mock }

func (m *ProdProductRepoMock) ListPublic(ctx context.Context, q repo.ProductListQuery) ([]model.Product, int64, error) {
	args := m.Called(ctx, q)
	items, _ := args.Get(0).([]model.Product)
	return items, args.Get(1).(int64), args.Error(2)
}

func (m *ProdProductRepoMock) SetActive(ctx context.Context, productID int64, isActive bool) error {
	panic("not used in ProductUsecase tests")
}

func (m *ProdProductRepoMock) FindByID(ctx context.Context, productID int64) (model.Product, error) {
	args := m.Called(ctx, productID)
	p, _ := args.Get(0).(model.Product)
	return p, args.Error(1)
}

func (m *ProdProductRepoMock) Create(ctx context.Context, p model.Product) (model.Product, error) {
	args := m.Called(ctx, p)
	created, _ := args.Get(0).(model.Product)
	return created, args.Error(1)
}

func (m *ProdProductRepoMock) Update(ctx context.Context, p model.Product) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *ProdProductRepoMock) SoftDelete(ctx context.Context, productID int64) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

type ProdInventoryRepoMock struct{ mock.Mock }

func (m *ProdInventoryRepoMock) SetStock(ctx context.Context, productID int64, newStock int64) error {
	args := m.Called(ctx, productID, newStock)
	return args.Error(0)
}

func (m *ProdInventoryRepoMock) DecreaseStockIfEnough(ctx context.Context, productID int64, qty int64) (bool, error) {
	panic("not used in ProductUsecase tests")
}

func (m *ProdInventoryRepoMock) IncreaseStock(ctx context.Context, productID int64, qty int64) error {
	panic("not used in ProductUsecase tests")
}

func (m *ProdInventoryRepoMock) CreateAdjustment(ctx context.Context, adj model.InventoryAdjustment) error {
	args := m.Called(ctx, adj)
	return args.Error(0)
}

type ProdAuditRepoMock struct{ mock.Mock }

func (m *ProdAuditRepoMock) Create(ctx context.Context, log model.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *ProdAuditRepoMock) List(ctx context.Context, filter repo.AuditLogFilter) ([]model.AuditLog, error) {
	panic("not used in ProductUsecase tests")
}

// =====================
// Public: List / Detail
// =====================

func TestProductUsecase_ListPublicProducts_InvalidPage(t *testing.T) {
	uc := usecase.NewProductUsecase(new(ProdProductRepoMock), new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	_, err := uc.ListPublicProducts(context.Background(), usecase.ListProductsInput{Page: 0, Limit: 20})
	assertErrContains(t, err, "invalid page")
}

func TestProductUsecase_ListPublicProducts_InvalidLimit(t *testing.T) {
	uc := usecase.NewProductUsecase(new(ProdProductRepoMock), new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	_, err := uc.ListPublicProducts(context.Background(), usecase.ListProductsInput{Page: 1, Limit: 101})
	assertErrContains(t, err, "invalid limit")
}

func TestProductUsecase_ListPublicProducts_Success(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	in := usecase.ListProductsInput{Page: 1, Limit: 20, Q: "coffee", Sort: "new"}
	q := repo.ProductListQuery{Page: 1, Limit: 20, Q: "coffee", Sort: "new"}

	items := []model.Product{
		{ID: 1, Name: "A", IsActive: true},
	}
	pRepo.On("ListPublic", mock.Anything, q).Return(items, int64(1), nil)

	out, err := uc.ListPublicProducts(ctx, in)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), out.Total)
	assert.Equal(t, 1, out.Page)
	assert.Equal(t, 20, out.Limit)
	assert.Equal(t, 1, len(out.Items))

	pRepo.AssertExpectations(t)
}

func TestProductUsecase_GetProductDetail_NotFound_WhenInactive(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("FindByID", mock.Anything, int64(1)).Return(model.Product{ID: 1, IsActive: false}, nil)

	_, err := uc.GetProductDetail(ctx, 1)
	assertErrContains(t, err, "not found")
}

func TestProductUsecase_GetProductDetail_NotFound_WhenRepoNotFound(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("FindByID", mock.Anything, int64(99)).Return(model.Product{}, repo.ErrNotFound)

	_, err := uc.GetProductDetail(ctx, 99)
	assertErrContains(t, err, "not found")
}

func TestProductUsecase_GetProductDetail_Success(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("FindByID", mock.Anything, int64(1)).Return(model.Product{ID: 1, IsActive: true}, nil)

	p, err := uc.GetProductDetail(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), p.ID)

	pRepo.AssertExpectations(t)
}

// =====================
// Admin: Product CRUD（A1〜A4相当）
// =====================

func TestProductUsecase_AdminCreateProduct_Unauthorized(t *testing.T) {
	uc := usecase.NewProductUsecase(new(ProdProductRepoMock), new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	_, err := uc.AdminCreateProduct(context.Background(), 0, usecase.AdminCreateProductInput{Name: "x", Price: 1, Stock: 1})
	assertErrContains(t, err, "unauthorized")
}

func TestProductUsecase_AdminCreateProduct_Validation(t *testing.T) {
	uc := usecase.NewProductUsecase(new(ProdProductRepoMock), new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	_, err := uc.AdminCreateProduct(context.Background(), 1, usecase.AdminCreateProductInput{Name: " ", Price: 1, Stock: 1})
	assertErrContains(t, err, "name required")
}

func TestProductUsecase_AdminCreateProduct_Success(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("Create", mock.Anything, mock.MatchedBy(func(p model.Product) bool {
		return p.Name == "Coffee" && p.Price == 100 && p.Stock == 10
	})).Return(model.Product{ID: 123}, nil)

	id, err := uc.AdminCreateProduct(ctx, 1, usecase.AdminCreateProductInput{
		Name:     " Coffee ",
		Price:    100,
		Stock:    10,
		IsActive: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)

	pRepo.AssertExpectations(t)
}

func TestProductUsecase_AdminUpdateProduct_NotFound(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("Update", mock.Anything, mock.AnythingOfType("model.Product")).Return(repo.ErrNotFound)

	err := uc.AdminUpdateProduct(ctx, 1, 999, usecase.AdminCreateProductInput{
		Name:  "X",
		Price: 1,
		Stock: 1,
	})
	assertErrContains(t, err, "not found")
}

func TestProductUsecase_AdminDeleteProduct_Success(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	uc := usecase.NewProductUsecase(pRepo, new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	pRepo.On("SoftDelete", mock.Anything, int64(1)).Return(nil)

	err := uc.AdminDeleteProduct(ctx, 1, 1)
	assert.NoError(t, err)

	pRepo.AssertExpectations(t)
}

// =====================
// Admin: Inventory update（S1/S3 + audit）
// =====================

func TestProductUsecase_AdminUpdateInventory_NegativeStock_S3(t *testing.T) {
	uc := usecase.NewProductUsecase(new(ProdProductRepoMock), new(ProdInventoryRepoMock), new(ProdAuditRepoMock))

	err := uc.AdminUpdateInventory(context.Background(), 1, 1, -1, "reason")
	assertErrContains(t, err, "stock must be >= 0")
}

// S1: 在庫更新 + 調整履歴 + 監査ログ
func TestProductUsecase_AdminUpdateInventory_Success_S1(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	iRepo := new(ProdInventoryRepoMock)
	aRepo := new(ProdAuditRepoMock)

	uc := usecase.NewProductUsecase(pRepo, iRepo, aRepo)

	// beforeの在庫を読む
	pRepo.On("FindByID", mock.Anything, int64(10)).Return(model.Product{ID: 10, Stock: 5, IsActive: true}, nil)

	// 在庫設定
	iRepo.On("SetStock", mock.Anything, int64(10), int64(12)).Return(nil)

	// 調整履歴
	iRepo.On("CreateAdjustment", mock.Anything, mock.MatchedBy(func(adj model.InventoryAdjustment) bool {
		// Delta = newStock - beforeStock
		return adj.ProductID == 10 && adj.AdminUserID == 1 && adj.Delta == 7 && strings.TrimSpace(adj.Reason) == "adjust"
	})).Return(nil)

	// 監査ログ
	aRepo.On("Create", mock.Anything, mock.MatchedBy(func(l model.AuditLog) bool {
		return l.ActorUserID == 1 &&
			l.Action == model.AuditActionUpdateStock &&
			l.ResourceType == model.AuditResourceProduct &&
			l.ResourceID == 10 &&
			l.BeforeJSON == `{"stock":5}` &&
			l.AfterJSON == `{"stock":12}`
	})).Return(nil)

	err := uc.AdminUpdateInventory(ctx, 1, 10, 12, " adjust ")
	assert.NoError(t, err)

	pRepo.AssertExpectations(t)
	iRepo.AssertExpectations(t)
	aRepo.AssertExpectations(t)
}

// 在庫更新でDBエラーなら 500
func TestProductUsecase_AdminUpdateInventory_DBError_OnSetStock(t *testing.T) {
	ctx := context.Background()

	pRepo := new(ProdProductRepoMock)
	iRepo := new(ProdInventoryRepoMock)
	aRepo := new(ProdAuditRepoMock)

	uc := usecase.NewProductUsecase(pRepo, iRepo, aRepo)

	pRepo.On("FindByID", mock.Anything, int64(10)).Return(model.Product{ID: 10, Stock: 5, IsActive: true}, nil)

	iRepo.On("SetStock", mock.Anything, int64(10), int64(12)).Return(errors.New("db down"))

	err := uc.AdminUpdateInventory(ctx, 1, 10, 12, "adjust")
	assertErrContains(t, err, "db error")
}
