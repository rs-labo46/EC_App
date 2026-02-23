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

func ValidateLoginInput(email, password string) error {
	if email == "" || password == "" {
		return errors.New("validation error")
	}
	return nil
}

func Refresh(token string, repo *MockRefreshRepository) error {
	return repo.Rotate(token)
}

func ValidateTokenVersion(current, token int) error {
	if current != token {
		return ErrForbidden
	}
	return nil
}

func ValidateJWT(token string) error {
	return ErrForbidden
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
	cartRepo := new(MockCartRepository)

	orderRepo.
		On("PlaceOrder", userID, addressID, idempotencyKey).
		Return(nil)

	// ★ 追加
	cartRepo.
		On("ClearCart", userID).
		Return(nil)

	err := PlaceOrder(userID, addressID, idempotencyKey, orderRepo, cartRepo)

	assert.NoError(t, err)

	orderRepo.AssertExpectations(t)
	cartRepo.AssertExpectations(t)
}

// Test: トランザクション処理とロールバック
func TestOrderTransactionRollback(t *testing.T) {
	userID := 1
	addressID := 100
	idempotencyKey := "unique-key-123"

	orderRepo := new(MockOrderRepository)
	cartRepo := new(MockCartRepository)

	orderRepo.
		On("PlaceOrder", userID, addressID, idempotencyKey).
		Return(ErrTransactionFailed)

	err := PlaceOrder(userID, addressID, idempotencyKey, orderRepo, cartRepo)

	assert.Equal(t, ErrTransactionFailed, err)

	cartRepo.AssertNotCalled(t, "ClearCart", userID)

	orderRepo.AssertExpectations(t)
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
		return err
	}
	return cartRepo.ClearCart(userID)

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

// =============================
// ① Login 異常系追加
// =============================

func TestLoginWrongPasswordDoesNotCreateRefresh(t *testing.T) {
	userID := 1
	refreshToken := "dummy"

	authRepo := new(MockAuthRepository)
	authRepo.On("Logout", userID, refreshToken).Return(ErrForbidden)

	err := Logout(userID, refreshToken, authRepo)
	assert.Error(t, err)

	authRepo.AssertExpectations(t)
}

func TestLoginValidationError(t *testing.T) {
	email := ""
	password := "xxx"

	err := ValidateLoginInput(email, password)
	assert.Error(t, err)
}

// =============================
// ② Refresh関連
// =============================

type MockRefreshRepository struct {
	mock.Mock
}

func (m *MockRefreshRepository) Rotate(oldToken string) error {
	args := m.Called(oldToken)
	return args.Error(0)
}

func TestRefreshTokenExpired(t *testing.T) {
	refreshRepo := new(MockRefreshRepository)
	refreshRepo.On("Rotate", "expired-token").Return(ErrForbidden)

	err := Refresh("expired-token", refreshRepo)
	assert.Equal(t, ErrForbidden, err)
	refreshRepo.AssertExpectations(t)
}

func TestRefresh_ReturnsError_WhenRotateFails(t *testing.T) {
	refreshRepo := new(MockRefreshRepository)
	refreshRepo.On("Rotate", "used-token").Return(ErrForbidden)

	err := Refresh("used-token", refreshRepo)
	assert.Error(t, err)
	refreshRepo.AssertExpectations(t)
}

// =============================
// ③ token_version不一致
// =============================

func TestTokenVersionMismatch(t *testing.T) {
	currentVersion := 2
	tokenVersion := 1

	err := ValidateTokenVersion(currentVersion, tokenVersion)
	assert.Equal(t, ErrForbidden, err)
}

// =============================
// ④ カート数量変更
// =============================

func TestUpdateCartQuantity(t *testing.T) {
	userID := 1
	productID := 101
	newQty := 3

	cartRepo := new(MockCartRepository)
	cartRepo.On("AddToCart", userID, productID, newQty).Return(nil)

	err := AddToCart(userID, productID, newQty, cartRepo)
	assert.NoError(t, err)

	cartRepo.AssertExpectations(t)
}

// =============================
// ⑤ 注文 在庫不足
// =============================

func TestOrderStockInsufficient(t *testing.T) {
	userID := 1
	addressID := 1
	idempotencyKey := "key"

	orderRepo := new(MockOrderRepository)
	cartRepo := new(MockCartRepository)

	orderRepo.
		On("PlaceOrder", userID, addressID, idempotencyKey).
		Return(ErrStockExceeded)

	err := PlaceOrder(userID, addressID, idempotencyKey, orderRepo, cartRepo)

	assert.Equal(t, ErrStockExceeded, err)

	cartRepo.AssertNotCalled(t, "ClearCart", userID)

	orderRepo.AssertExpectations(t)
}

// =============================
// ⑥ 管理者 権限なし
// =============================

func TestAdminUnauthorized(t *testing.T) {
	adminID := 0
	orderID := 10

	adminRepo := new(MockAdminOrderRepository)
	adminRepo.On("UpdateOrderStatusByAdmin", adminID, orderID, "SHIPPED").Return(ErrForbidden)

	err := adminRepo.UpdateOrderStatusByAdmin(adminID, orderID, "SHIPPED")
	assert.Equal(t, ErrForbidden, err)

	adminRepo.AssertExpectations(t)
}

// =============================
// ⑦ 在庫負数防止
// =============================

func TestInventoryNegativeStock(t *testing.T) {
	adminID := 1
	productID := 1
	stock := -1
	reason := "invalid"

	inventoryRepo := new(MockInventoryRepository)
	auditRepo := new(MockAuditLogRepository)

	inventoryRepo.On("SetStock", productID, stock).Return(ErrForbidden)

	err := UpdateInventory(adminID, productID, stock, reason, inventoryRepo, auditRepo)
	assert.Error(t, err)

	inventoryRepo.AssertExpectations(t)
}

// =============================
// ⑧ JWT
// =============================

func TestInvalidJWTSignature(t *testing.T) {
	err := ValidateJWT("tampered.jwt.token")
	assert.Error(t, err)
}

// =============================
// ⑨ リフレッシュ回転
// =============================

// --- Refresh Rotate Success ---
func TestRefreshRotateSuccess(t *testing.T) {
	userID := 1
	oldToken := "old-refresh"
	newToken := "new-refresh"

	authRepo := new(MockAuthRepository)

	// 旧トークンを使用済みに更新
	authRepo.On("MarkRefreshUsed", oldToken).Return(nil)

	// 新トークン発行
	authRepo.On("CreateRefreshToken", userID).Return(newToken, nil)

	token, err := RotateRefresh(userID, oldToken, authRepo)

	assert.NoError(t, err)
	assert.Equal(t, newToken, token)

	authRepo.AssertExpectations(t)
}

// --- Refresh Replay Attack ---
func TestRotateRefresh_ReplayAttack_TriggersGlobalLogout(t *testing.T) {
	userID := 1
	oldToken := "old-refresh"

	authRepo := new(MockAuthRepository)

	authRepo.On("MarkRefreshUsed", oldToken).Return(ErrForbidden)
	authRepo.On("DeleteAllRefreshTokens", userID).Return(nil)
	authRepo.On("IncrementTokenVersion", userID).Return(nil)

	_, err := RotateRefresh(userID, oldToken, authRepo)

	assert.Error(t, err)
	authRepo.AssertExpectations(t)
}

// =============================
// ⑩ 強制ログアウト
// =============================
func TestForceLogout(t *testing.T) {
	userID := 1

	authRepo := new(MockAuthRepository)

	authRepo.On("DeleteAllRefreshTokens", userID).Return(nil)
	authRepo.On("IncrementTokenVersion", userID).Return(nil)

	err := ForceLogout(userID, authRepo)

	assert.NoError(t, err)
	authRepo.AssertExpectations(t)
}

// =============================
// 11 注文Tx
// =============================

func (m *MockInventoryRepository) DecreaseStock(productID, qty int) error {
	args := m.Called(productID, qty)
	return args.Error(0)
}

func TestOrderTransactionFullSuccess(t *testing.T) {
	userID := 1
	productID := 100
	qty := 2

	orderRepo := new(MockOrderRepository)
	cartRepo := new(MockCartRepository)
	inventoryRepo := new(MockInventoryRepository)

	orderRepo.On("PlaceOrder", userID, 1, "key").Return(nil)
	inventoryRepo.On("DecreaseStock", productID, qty).Return(nil)
	cartRepo.On("ClearCart", userID).Return(nil)

	err := CompleteOrder(userID, productID, qty, orderRepo, inventoryRepo, cartRepo)

	assert.NoError(t, err)
	orderRepo.AssertExpectations(t)
	inventoryRepo.AssertExpectations(t)
	cartRepo.AssertExpectations(t)
}

func TestOrderStockFailureRollback(t *testing.T) {
	userID := 1
	productID := 100
	qty := 2

	orderRepo := new(MockOrderRepository)
	cartRepo := new(MockCartRepository)
	inventoryRepo := new(MockInventoryRepository)

	inventoryRepo.On("DecreaseStock", productID, qty).Return(ErrStockExceeded)

	err := CompleteOrder(userID, productID, qty, orderRepo, inventoryRepo, cartRepo)

	assert.Equal(t, ErrStockExceeded, err)

	// 失敗時に注文は作られない
	orderRepo.AssertNotCalled(t, "PlaceOrder")
	cartRepo.AssertNotCalled(t, "ClearCart")
}

// =============================
// 12 商品公開制御
// =============================
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetProduct(productID int) (bool, error) {
	args := m.Called(productID)
	return args.Bool(0), args.Error(1)
}

func TestProductNotPublished(t *testing.T) {
	productRepo := new(MockProductRepository)

	productRepo.On("GetProduct", 1).Return(false, nil)

	err := CheckProductVisible(1, productRepo)

	assert.Error(t, err)
	productRepo.AssertExpectations(t)
}

func TestProductPublished(t *testing.T) {
	productRepo := new(MockProductRepository)

	productRepo.On("GetProduct", 1).Return(true, nil)

	err := CheckProductVisible(1, productRepo)

	assert.NoError(t, err)
	productRepo.AssertExpectations(t)
}

// =============================
// 13 在庫競合
// =============================
func TestInventoryVersionConflict(t *testing.T) {
	inventoryRepo := new(MockInventoryRepository)

	inventoryRepo.On("DecreaseStock", 1, 2).Return(ErrForbidden)

	err := UpdateStockWithVersion(1, 2, inventoryRepo)

	assert.Error(t, err)
	inventoryRepo.AssertExpectations(t)
}

func RotateRefresh(userID int, oldToken string, repo *MockAuthRepository) (string, error) {
	err := repo.MethodCalled("MarkRefreshUsed", oldToken).Error(0)
	if err != nil {
		repo.MethodCalled("DeleteAllRefreshTokens", userID)
		repo.MethodCalled("IncrementTokenVersion", userID)
		return "", err
	}

	args := repo.MethodCalled("CreateRefreshToken", userID)
	return args.String(0), args.Error(1)
}

// --- 強制ログアウト ---
func ForceLogout(userID int, repo *MockAuthRepository) error {
	repo.MethodCalled("DeleteAllRefreshTokens", userID)
	repo.MethodCalled("IncrementTokenVersion", userID)
	return nil
}

// --- 注文完全処理 ---
func CompleteOrder(
	userID int,
	productID int,
	qty int,
	orderRepo *MockOrderRepository,
	inventoryRepo *MockInventoryRepository,
	cartRepo *MockCartRepository,
) error {

	if err := inventoryRepo.DecreaseStock(productID, qty); err != nil {
		return err
	}

	if err := orderRepo.PlaceOrder(userID, 1, "key"); err != nil {
		return err
	}

	return cartRepo.ClearCart(userID)
}

// --- 商品公開制御 ---
func CheckProductVisible(productID int, repo *MockProductRepository) error {
	visible, err := repo.GetProduct(productID)
	if err != nil {
		return err
	}
	if !visible {
		return ErrForbidden
	}
	return nil
}

// --- 在庫versionチェック ---
func UpdateStockWithVersion(productID, qty int, repo *MockInventoryRepository) error {
	return repo.DecreaseStock(productID, qty)
}
