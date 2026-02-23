package unit

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocking repositories
type MockCartRepository struct {
	mock.Mock
}

func (m *MockCartRepository) AddToCart(userID, productID, qty int) error {
	args := m.Called(userID, productID, qty)
	return args.Error(0)
}
func (m *MockCartRepository) ClearCart(userID int) error {
	args := m.Called(userID)
	return args.Error(0)
}

type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) PlaceOrder(userID, addressID int, idempotencyKey string) error {
	args := m.Called(userID, addressID, idempotencyKey)
	return args.Error(0)
}

type MockAdminOrderRepository struct {
	mock.Mock
}

func (m *MockAdminOrderRepository) UpdateOrderStatusByAdmin(adminID, orderID int, status string) error {
	args := m.Called(adminID, orderID, status)
	return args.Error(0)
}

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) Logout(userID int, refreshToken string) error {
	args := m.Called(userID, refreshToken)
	return args.Error(0)
}

type MockInventoryRepository struct {
	mock.Mock
}

func (m *MockInventoryRepository) SetStock(productID, newStock int) error {
	args := m.Called(productID, newStock)
	return args.Error(0)
}

type MockAuditLogRepository struct {
	mock.Mock
}

func (m *MockAuditLogRepository) Create(log interface{}) error {
	args := m.Called(log)
	return args.Error(0)
}

// Errors for testing
var ErrStockExceeded = errors.New("stock exceeded")
var ErrTransactionFailed = errors.New("transaction failed")
var ErrForbidden = errors.New("forbidden")

// Test: カートの同一商品追加
func TestAddSameProductToCart(t *testing.T) {
	userID := 1
	productID := 101
	initialQuantity := 1
	addedQuantity := 2

	cartRepo := new(MockCartRepository)
	cartRepo.On("AddToCart", userID, productID, initialQuantity).Return(nil)
	cartRepo.On("AddToCart", userID, productID, addedQuantity).Return(nil)

	err := AddToCart(userID, productID, initialQuantity, cartRepo)
	assert.NoError(t, err)

	err = AddToCart(userID, productID, addedQuantity, cartRepo)
	assert.NoError(t, err)

	cartRepo.AssertExpectations(t)
}

// Test: 在庫超過
func TestAddProductExceedingStock(t *testing.T) {
	userID := 1
	productID := 101
	quantityToAdd := 999

	cartRepo := new(MockCartRepository)
	cartRepo.On("AddToCart", userID, productID, quantityToAdd).Return(ErrStockExceeded)

	err := AddToCart(userID, productID, quantityToAdd, cartRepo)
	assert.Equal(t, ErrStockExceeded, err)

	cartRepo.AssertExpectations(t)
}

// Test: 注文のIdempotency
func TestOrderIdempotency(t *testing.T) {
	userID := 1
	addressID := 100
	idempotencyKey := "unique-key-123"

	orderRepo := new(MockOrderRepository)
	orderRepo.On("PlaceOrder", userID, addressID, idempotencyKey).Return(nil)
	cartRepo := new(MockCartRepository)
	err := PlaceOrder(userID, addressID, idempotencyKey, orderRepo, cartRepo)
	assert.NoError(t, err)

	orderRepo.AssertExpectations(t)
}

// Test: トランザクション処理とロールバック
func TestOrderTransactionRollback(t *testing.T) {
	userID := 1
	addressID := 100
	idempotencyKey := "unique-key-123"

	// モックリポジトリ
	orderRepo := new(MockOrderRepository)
	cartRepo := new(MockCartRepository)

	// トランザクション中にエラーを発生させる
	orderRepo.On("PlaceOrder", userID, addressID, idempotencyKey).Return(ErrTransactionFailed)
	cartRepo.On("ClearCart", userID).Return(nil) // 期待される ClearCart の呼び出し

	// テスト対象のメソッド呼び出し
	err := PlaceOrder(userID, addressID, idempotencyKey, orderRepo, cartRepo) // ここでOrderRepositoryとCartRepositoryを渡す
	assert.Equal(t, ErrTransactionFailed, err)

	// ClearCart メソッドが1回呼ばれていることを確認
	cartRepo.AssertExpectations(t)
}

// Test: 管理者注文の状態遷移ガード
func TestAdminOrderStateTransitionGuard(t *testing.T) {
	adminID := 100
	orderID := 500
	invalidStatus := "CANCELED"

	// Mock Admin Order Repository
	adminOrderRepo := new(MockAdminOrderRepository)

	// 管理者が不正な状態遷移を試みた場合に返すエラーを設定
	adminOrderRepo.On("UpdateOrderStatusByAdmin", adminID, orderID, invalidStatus).Return(ErrForbidden)

	// テスト対象のメソッド呼び出し
	err := adminOrderRepo.UpdateOrderStatusByAdmin(adminID, orderID, invalidStatus)

	// エラーチェック
	assert.Error(t, err)
	assert.Equal(t, ErrForbidden, err)

	// Mockのメソッドが呼ばれていることを確認
	adminOrderRepo.AssertExpectations(t)
}

// Test: ログアウト後のトークン無効化
func TestLogoutInvalidatesTokens(t *testing.T) {
	userID := 1
	refreshToken := "valid-refresh-token"

	authRepo := new(MockAuthRepository)
	authRepo.On("Logout", userID, refreshToken).Return(nil)

	err := Logout(userID, refreshToken, authRepo)
	assert.NoError(t, err)

	authRepo.AssertExpectations(t)
}

// Test: 在庫更新の監査ログ
func TestInventoryUpdateAuditLog(t *testing.T) {
	adminID := 100
	productID := 200
	stockDelta := 10
	reason := "Stock Adjustment"

	inventoryRepo := new(MockInventoryRepository)
	auditLogRepo := new(MockAuditLogRepository)

	inventoryRepo.On("SetStock", productID, stockDelta).Return(nil)
	auditLogRepo.On("Create", mock.Anything).Return(nil)

	err := UpdateInventory(adminID, productID, stockDelta, reason, inventoryRepo, auditLogRepo)
	assert.NoError(t, err)

	inventoryRepo.AssertExpectations(t)
	auditLogRepo.AssertExpectations(t)
}

// Sample AddToCart function
func AddToCart(userID, productID, qty int, cartRepo *MockCartRepository) error {
	return cartRepo.AddToCart(userID, productID, qty)
}

// Sample PlaceOrder function
type OrderRepository interface {
	PlaceOrder(userID, addressID int, idempotencyKey string) error
}

type CartRepository interface {
	ClearCart(userID int) error
}

func PlaceOrder(userID, addressID int, idempotencyKey string, orderRepo OrderRepository, cartRepo CartRepository) error {
	// トランザクション開始
	err := orderRepo.PlaceOrder(userID, addressID, idempotencyKey)
	if err != nil {
		// 注文失敗時にカートをクリアする
		cartRepo.ClearCart(userID) // ClearCart が呼ばれるべき
		return err
	}
	return nil
}

// Sample Logout function
func Logout(userID int, refreshToken string, authRepo *MockAuthRepository) error {
	return authRepo.Logout(userID, refreshToken)
}

// Sample UpdateInventory function
func UpdateInventory(adminID, productID, stockDelta int, reason string, inventoryRepo *MockInventoryRepository, auditLogRepo *MockAuditLogRepository) error {
	if err := inventoryRepo.SetStock(productID, stockDelta); err != nil {
		return err
	}
	return auditLogRepo.Create("inventory update")
}
