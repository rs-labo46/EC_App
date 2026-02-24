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
// TxManager / TxRepos mocks
// =====================

// AdminTxManagerMock は WithinTx の中で渡す repos を固定して unit テストを回す
type AdminTxManagerMock struct {
	mock.Mock
	Repos repo.TxRepos
}

func (m *AdminTxManagerMock) WithinTx(ctx context.Context, fn func(r repo.TxRepos) error) error {
	// 呼ばれた事実だけ記録（ctxの具体値は問わない）
	m.Called(ctx)
	return fn(m.Repos)
}

type AdminTxReposMock struct {
	orders     repo.OrderRepository
	orderItems repo.OrderItemRepository
	inventory  repo.InventoryRepository

	// AdminOrderUsecase では使わないが TxRepos interface を満たすために保持
	carts     repo.CartRepository
	cartItems repo.CartItemRepository
	products  repo.ProductRepository
}

func (r *AdminTxReposMock) Orders() repo.OrderRepository         { return r.orders }
func (r *AdminTxReposMock) OrderItems() repo.OrderItemRepository { return r.orderItems }
func (r *AdminTxReposMock) Inventory() repo.InventoryRepository  { return r.inventory }
func (r *AdminTxReposMock) Carts() repo.CartRepository           { return r.carts }
func (r *AdminTxReposMock) CartItems() repo.CartItemRepository   { return r.cartItems }
func (r *AdminTxReposMock) Products() repo.ProductRepository     { return r.products }

// =====================
// Repository mocks (Admin向け：衝突回避)
// =====================

type AdminOrderRepoMock struct{ mock.Mock }

func (m *AdminOrderRepoMock) FindByID(ctx context.Context, orderID int64) (model.Order, error) {
	args := m.Called(ctx, orderID)
	o, _ := args.Get(0).(model.Order)
	return o, args.Error(1)
}

func (m *AdminOrderRepoMock) ListByUserID(ctx context.Context, userID int64, page int, limit int) ([]model.Order, int64, error) {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminOrderRepoMock) Create(ctx context.Context, order model.Order) (int64, error) {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminOrderRepoMock) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *AdminOrderRepoMock) FindByIdempotencyKey(ctx context.Context, userID int64, key string) (model.Order, bool, error) {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminOrderRepoMock) ListAdmin(ctx context.Context, f repo.AdminOrderListFilter) ([]model.Order, int64, error) {
	args := m.Called(ctx, f)
	orders, _ := args.Get(0).([]model.Order)
	return orders, args.Get(1).(int64), args.Error(2)
}

type AdminOrderItemRepoMock struct{ mock.Mock }

func (m *AdminOrderItemRepoMock) CreateBulk(ctx context.Context, orderID int64, items []model.OrderItem) error {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminOrderItemRepoMock) ListByOrderID(ctx context.Context, orderID int64) ([]model.OrderItem, error) {
	args := m.Called(ctx, orderID)
	items, _ := args.Get(0).([]model.OrderItem)
	return items, args.Error(1)
}

type AdminInventoryRepoMock struct{ mock.Mock }

func (m *AdminInventoryRepoMock) SetStock(ctx context.Context, productID int64, newStock int64) error {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminInventoryRepoMock) DecreaseStockIfEnough(ctx context.Context, productID int64, qty int64) (bool, error) {
	panic("not used in AdminOrderUsecase tests")
}

func (m *AdminInventoryRepoMock) IncreaseStock(ctx context.Context, productID int64, qty int64) error {
	args := m.Called(ctx, productID, qty)
	return args.Error(0)
}

func (m *AdminInventoryRepoMock) CreateAdjustment(ctx context.Context, adjustment model.InventoryAdjustment) error {
	panic("not used in AdminOrderUsecase tests")
}

type AdminAuditRepoMock struct{ mock.Mock }

func (m *AdminAuditRepoMock) Create(ctx context.Context, log model.AuditLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *AdminAuditRepoMock) List(ctx context.Context, filter repo.AuditLogFilter) ([]model.AuditLog, error) {
	panic("not used in AdminOrderUsecase tests")
}

// =====================
// Helper: error contains（HTTPErrorの実装詳細に依存しない）
// =====================

func assertErrContains(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if assert.Error(t, err) {
		assert.True(t, strings.Contains(err.Error(), wantSubstr), "err=%q want contains %q", err.Error(), wantSubstr)
	}
}

// =====================
// List tests
// =====================

func TestAdminOrderUsecase_List_InvalidPage(t *testing.T) {
	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	outs, err := uc.List(context.Background(), repo.AdminOrderListFilter{Page: 0, Limit: 20})
	assert.Equal(t, 0, len(outs))
	assertErrContains(t, err, "invalid page")
}

func TestAdminOrderUsecase_List_InvalidLimit(t *testing.T) {
	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	outs, err := uc.List(context.Background(), repo.AdminOrderListFilter{Page: 1, Limit: 0})
	assert.Equal(t, 0, len(outs))
	assertErrContains(t, err, "invalid limit")
}

func TestAdminOrderUsecase_List_Success_CallsItemsPerOrder(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)
	itemsRepo := new(AdminOrderItemRepoMock)
	invRepo := new(AdminInventoryRepoMock)

	tx.Repos = &AdminTxReposMock{
		orders:     ordersRepo,
		orderItems: itemsRepo,
		inventory:  invRepo,
	}
	tx.On("WithinTx", mock.Anything).Return(nil)

	f := repo.AdminOrderListFilter{Page: 1, Limit: 20}

	orders := []model.Order{
		{ID: 10, Status: model.OrderStatusPending},
		{ID: 11, Status: model.OrderStatusPaid},
	}

	ordersRepo.On("ListAdmin", mock.Anything, f).Return(orders, int64(2), nil)
	itemsRepo.On("ListByOrderID", mock.Anything, int64(10)).Return([]model.OrderItem{}, nil)
	itemsRepo.On("ListByOrderID", mock.Anything, int64(11)).Return([]model.OrderItem{}, nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	outs, err := uc.List(ctx, f)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(outs))

	tx.AssertExpectations(t)
	ordersRepo.AssertExpectations(t)
	itemsRepo.AssertExpectations(t)
}

// =====================
// UpdateStatus tests
// =====================

func TestAdminOrderUsecase_UpdateStatus_UnauthorizedActor(t *testing.T) {
	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)
	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(context.Background(), 0, 1, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assertErrContains(t, err, "unauthorized")
}

func TestAdminOrderUsecase_UpdateStatus_InvalidOrderID(t *testing.T) {
	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)
	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(context.Background(), 1, 0, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assertErrContains(t, err, "invalid id")
}

func TestAdminOrderUsecase_UpdateStatus_InvalidStatus(t *testing.T) {
	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)
	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(context.Background(), 1, 1, usecase.AdminUpdateOrderStatusInput{Status: "XXX"})
	assertErrContains(t, err, "invalid status")
}

func TestAdminOrderUsecase_UpdateStatus_NotFound(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)

	tx.Repos = &AdminTxReposMock{orders: ordersRepo}
	tx.On("WithinTx", mock.Anything).Return(nil)

	orderID := int64(99)

	ordersRepo.On("FindByID", mock.Anything, orderID).Return(model.Order{}, repo.ErrNotFound)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, 1, orderID, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assertErrContains(t, err, "not found")

	ordersRepo.AssertExpectations(t)
}

func TestAdminOrderUsecase_UpdateStatus_SameStatus_NoOp(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)

	tx.Repos = &AdminTxReposMock{orders: ordersRepo}
	tx.On("WithinTx", mock.Anything).Return(nil)

	orderID := int64(1)

	ordersRepo.On("FindByID", mock.Anything, orderID).Return(model.Order{
		ID:     orderID,
		Status: model.OrderStatusPaid,
	}, nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, 1, orderID, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assert.NoError(t, err)

	ordersRepo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
	audit.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAdminOrderUsecase_UpdateStatus_CannotChangeCanceled(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)

	tx.Repos = &AdminTxReposMock{orders: ordersRepo}
	tx.On("WithinTx", mock.Anything).Return(nil)

	ordersRepo.On("FindByID", mock.Anything, int64(1)).Return(model.Order{
		ID:     1,
		Status: model.OrderStatusCanceled,
	}, nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, 1, 1, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assertErrContains(t, err, "cannot change canceled order")
}

func TestAdminOrderUsecase_UpdateStatus_CannotChangeShipped(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)

	tx.Repos = &AdminTxReposMock{orders: ordersRepo}
	tx.On("WithinTx", mock.Anything).Return(nil)

	ordersRepo.On("FindByID", mock.Anything, int64(1)).Return(model.Order{
		ID:     1,
		Status: model.OrderStatusShipped,
	}, nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, 1, 1, usecase.AdminUpdateOrderStatusInput{Status: "PAID"})
	assertErrContains(t, err, "cannot change shipped order")
}

// cancel: PENDING/PAID -> CANCELED のとき在庫戻し + audit
func TestAdminOrderUsecase_UpdateStatus_Cancel_RestoresStock_And_Audits(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)
	itemsRepo := new(AdminOrderItemRepoMock)
	invRepo := new(AdminInventoryRepoMock)

	tx.Repos = &AdminTxReposMock{
		orders:     ordersRepo,
		orderItems: itemsRepo,
		inventory:  invRepo,
	}
	tx.On("WithinTx", mock.Anything).Return(nil)

	adminID := int64(999)
	orderID := int64(50)

	ordersRepo.On("FindByID", mock.Anything, orderID).Return(model.Order{
		ID:     orderID,
		Status: model.OrderStatusPaid,
	}, nil)

	items := []model.OrderItem{
		{OrderID: orderID, ProductID: 100, Quantity: 2},
		{OrderID: orderID, ProductID: 101, Quantity: 1},
	}
	itemsRepo.On("ListByOrderID", mock.Anything, orderID).Return(items, nil)

	invRepo.On("IncreaseStock", mock.Anything, int64(100), int64(2)).Return(nil)
	invRepo.On("IncreaseStock", mock.Anything, int64(101), int64(1)).Return(nil)

	ordersRepo.On("UpdateStatus", mock.Anything, orderID, model.OrderStatus("CANCELED")).Return(nil)

	audit.On("Create", mock.Anything, mock.MatchedBy(func(a model.AuditLog) bool {
		// CreatedAt は now なので見ない
		if a.ActorUserID != adminID {
			return false
		}
		if a.Action != model.AuditActionUpdateOrderStatus {
			return false
		}
		if a.ResourceType != model.AuditResourceOrder {
			return false
		}
		if a.ResourceID != orderID {
			return false
		}
		if a.BeforeJSON != `{"status":"PAID"}` {
			return false
		}
		if a.AfterJSON != `{"status":"CANCELED"}` {
			return false
		}
		return true
	})).Return(nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, adminID, orderID, usecase.AdminUpdateOrderStatusInput{Status: "CANCELED"})
	assert.NoError(t, err)

	ordersRepo.AssertExpectations(t)
	itemsRepo.AssertExpectations(t)
	invRepo.AssertExpectations(t)
	audit.AssertExpectations(t)
}

// shipped: PENDING -> SHIPPED は在庫戻しなし + audit
func TestAdminOrderUsecase_UpdateStatus_Shipped_Audits_NoInventory(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)
	itemsRepo := new(AdminOrderItemRepoMock)
	invRepo := new(AdminInventoryRepoMock)

	tx.Repos = &AdminTxReposMock{
		orders:     ordersRepo,
		orderItems: itemsRepo,
		inventory:  invRepo,
	}
	tx.On("WithinTx", mock.Anything).Return(nil)

	adminID := int64(1)
	orderID := int64(60)

	ordersRepo.On("FindByID", mock.Anything, orderID).Return(model.Order{
		ID:     orderID,
		Status: model.OrderStatusPending,
	}, nil)

	ordersRepo.On("UpdateStatus", mock.Anything, orderID, model.OrderStatus("SHIPPED")).Return(nil)

	audit.On("Create", mock.Anything, mock.MatchedBy(func(a model.AuditLog) bool {
		return a.ActorUserID == adminID &&
			a.ResourceID == orderID &&
			a.BeforeJSON == `{"status":"PENDING"}` &&
			a.AfterJSON == `{"status":"SHIPPED"}`
	})).Return(nil)

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, adminID, orderID, usecase.AdminUpdateOrderStatusInput{Status: "SHIPPED"})
	assert.NoError(t, err)

	// cancel じゃないので在庫戻しは呼ばれない
	itemsRepo.AssertNotCalled(t, "ListByOrderID", mock.Anything, mock.Anything)
	invRepo.AssertNotCalled(t, "IncreaseStock", mock.Anything, mock.Anything, mock.Anything)

	ordersRepo.AssertExpectations(t)
	audit.AssertExpectations(t)
}

func TestAdminOrderUsecase_UpdateStatus_DBError_OnUpdate(t *testing.T) {
	ctx := context.Background()

	tx := new(AdminTxManagerMock)
	audit := new(AdminAuditRepoMock)

	ordersRepo := new(AdminOrderRepoMock)

	tx.Repos = &AdminTxReposMock{orders: ordersRepo}
	tx.On("WithinTx", mock.Anything).Return(nil)

	orderID := int64(70)

	ordersRepo.On("FindByID", mock.Anything, orderID).Return(model.Order{
		ID:     orderID,
		Status: model.OrderStatusPaid,
	}, nil)

	ordersRepo.On("UpdateStatus", mock.Anything, orderID, model.OrderStatus("SHIPPED")).Return(errors.New("db down"))

	uc := usecase.NewAdminOrderUsecase(tx, audit)

	err := uc.UpdateStatus(ctx, 1, orderID, usecase.AdminUpdateOrderStatusInput{Status: "SHIPPED"})
	assertErrContains(t, err, "db error")
}
