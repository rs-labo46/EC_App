package unit

import (
	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/usecase"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// =====================
// Mock: UserRepository
// =====================

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	u, _ := args.Get(0).(*model.User)
	return u, args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id int64) (*model.User, error) {
	args := m.Called(ctx, id)
	u, _ := args.Get(0).(*model.User)
	return u, args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) IncrementTokenVersion(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// =====================
// Mock: AuthValidator
// =====================

type MockAuthValidator struct {
	mock.Mock
}

func (m *MockAuthValidator) ValidateRegister(ctx context.Context, email string, password string) error {
	args := m.Called(ctx, email, password)
	return args.Error(0)
}

func (m *MockAuthValidator) ValidateLogin(ctx context.Context, email string, password string) error {
	args := m.Called(ctx, email, password)
	return args.Error(0)
}

func (m *MockAuthValidator) ValidateRefresh(ctx context.Context, refreshToken string, userAgent string) error {
	args := m.Called(ctx, refreshToken, userAgent)
	return args.Error(0)
}

func (m *MockAuthValidator) ValidateLogout(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockAuthValidator) ValidateForceLogout(ctx context.Context, targetUserID int64) error {
	args := m.Called(ctx, targetUserID)
	return args.Error(0)
}

// =====================
// Mock: RefreshTokenRepository
// =====================

type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) Create(ctx context.Context, token model.RefreshToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (model.RefreshToken, bool, error) {
	args := m.Called(ctx, tokenHash)
	rt, _ := args.Get(0).(model.RefreshToken)
	return rt, args.Bool(1), args.Error(2)
}

func (m *MockRefreshTokenRepository) MarkUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteByID(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	// ★ 重要：引数をそのまま渡す（これがズレると Unexpected Method Call になる）
	args := m.Called(ctx, now)
	return args.Get(0).(int64), args.Error(1)
}

// =====================
// Helper
// =====================

func mustHash(t *testing.T, plain string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt failed: %v", err)
	}
	return string(b)
}

func newAuthUC(userRepo *MockUserRepository, rtRepo *MockRefreshTokenRepository, v *MockAuthValidator) *usecase.AuthUsecase {
	// JWTSecret は Login/Refresh で必須
	cfg := config.Config{JWTSecret: "test-secret"}
	return usecase.NewAuthUsecase(cfg, userRepo, rtRepo, v)
}

// =====================
// Register（参考：ユースケース単体）
// =====================

func TestAuthUsecase_Register_Success(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository) // Registerでは使わないがDI上必要
	v := new(MockAuthValidator)

	email := "user@test.com"
	pass := "CorrectPW"

	v.On("ValidateRegister", mock.Anything, email, pass).Return(nil)

	userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		// 保存されるユーザーが最低限正しい形かを見る
		return u.Email == email && u.IsActive && u.TokenVersion == 0 && u.PasswordHash != ""
	})).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	resp, err := u.Register(ctx, usecase.AuthRegisterRequest{Email: email, Password: pass})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, email, resp.User.Email)

	userRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// =====================
// Login（方針② login: A1/A2/A3 + 停止ユーザー）
// =====================

func TestAuthUsecase_Login_Success_A1(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	email := "user@test.com"
	pass := "CorrectPW"

	// DeleteExpired は失敗してもログイン継続。呼ばれたら 0 件でOK
	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)

	v.On("ValidateLogin", mock.Anything, email, pass).Return(nil)

	userRepo.On("FindByEmail", mock.Anything, email).Return(&model.User{
		ID:           1,
		Email:        email,
		PasswordHash: mustHash(t, pass),
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     true,
	}, nil)

	// last_login 更新は失敗しても継続なので、呼ばれてもOK
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	// refresh 保存が呼ばれる（中身はランダムなので型だけ確認）
	rtRepo.On("Create", mock.Anything, mock.AnythingOfType("model.RefreshToken")).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Login(ctx, usecase.AuthLoginRequest{Email: email, Password: pass}, "UserAgent", "127.0.0.1")
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NotEmpty(t, res.Body.Token.AccessToken)
	assert.Greater(t, res.Body.Token.ExpiresIn, 0)
	assert.Equal(t, 0, res.Body.Token.TokenVersion)

	assert.NotEmpty(t, res.RefreshTokenPlain)
	assert.NotEmpty(t, res.CsrfTokenPlain)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② A2: PW違い => 401 / refresh増えない
func TestAuthUsecase_Login_WrongPassword_A2(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	email := "user@test.com"
	pass := "WrongPW"

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateLogin", mock.Anything, email, pass).Return(nil)

	// DB上のhashは正しいパスワードのもの
	userRepo.On("FindByEmail", mock.Anything, email).Return(&model.User{
		ID:           1,
		Email:        email,
		PasswordHash: mustHash(t, "CorrectPW"),
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     true,
	}, nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Login(ctx, usecase.AuthLoginRequest{Email: email, Password: pass}, "UA", "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrUnauthorized)

	// refresh token は作られない
	rtRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② A3: email空 => 400（validatorが ErrValidation を返す想定）
func TestAuthUsecase_Login_ValidationError_A3(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)

	v.On("ValidateLogin", mock.Anything, "", "xxx").Return(usecase.ErrValidation)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Login(ctx, usecase.AuthLoginRequest{Email: "", Password: "xxx"}, "UA", "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrValidation)

	// validatorで落ちるので repo は呼ばれない
	userRepo.AssertNotCalled(t, "FindByEmail", mock.Anything, mock.Anything)

	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 停止ユーザー => forbidden（方針② login 前提の is_active）
func TestAuthUsecase_Login_InactiveUser(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	email := "user@test.com"
	pass := "CorrectPW"

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateLogin", mock.Anything, email, pass).Return(nil)

	userRepo.On("FindByEmail", mock.Anything, email).Return(&model.User{
		ID:           1,
		Email:        email,
		PasswordHash: mustHash(t, pass),
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     false,
	}, nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Login(ctx, usecase.AuthLoginRequest{Email: email, Password: pass}, "UA", "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrForbidden)

	// refresh作成されない
	rtRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// =====================
// Refresh（方針② refresh: R1/R2/R4/R5）
// =====================

// 方針② R1: 正常（旧token used_at 更新 + 新token追加）
func TestAuthUsecase_Refresh_Success_R1(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	refreshPlain := "refresh-plain"
	ua := "UA"
	userID := int64(1)

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateRefresh", mock.Anything, refreshPlain, ua).Return(nil)

	rtRepo.On("FindByHash", mock.Anything, mock.AnythingOfType("string")).Return(model.RefreshToken{
		ID:        "rt-old",
		UserID:    userID,
		UserAgent: ua,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		UsedAt:    nil,
	}, true, nil)

	userRepo.On("FindByID", mock.Anything, userID).Return(&model.User{
		ID:           userID,
		Email:        "user@test.com",
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     true,
	}, nil)

	rtRepo.On("MarkUsed", mock.Anything, "rt-old").Return(nil)
	rtRepo.On("Create", mock.Anything, mock.AnythingOfType("model.RefreshToken")).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Refresh(ctx, refreshPlain, ua, "127.0.0.1")
	assert.NoError(t, err)
	assert.NotNil(t, res)

	assert.NotEmpty(t, res.Body.AccessToken)
	assert.Greater(t, res.Body.ExpiresIn, 0)
	assert.Equal(t, 0, res.Body.TokenVersion)

	assert.NotEmpty(t, res.RefreshTokenPlain)
	assert.NotEmpty(t, res.CsrfTokenPlain)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② R2: 期限切れ => DeleteByID + 401
func TestAuthUsecase_Refresh_Expired_R2(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	refreshPlain := "expired"
	ua := "UA"

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateRefresh", mock.Anything, refreshPlain, ua).Return(nil)

	rtRepo.On("FindByHash", mock.Anything, mock.AnythingOfType("string")).Return(model.RefreshToken{
		ID:        "rt-exp",
		UserID:    1,
		UserAgent: ua,
		ExpiresAt: time.Now().Add(-1 * time.Minute),
		UsedAt:    nil,
	}, true, nil)

	rtRepo.On("DeleteByID", mock.Anything, "rt-exp").Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Refresh(ctx, refreshPlain, ua, "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrUnauthorized)

	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② R4: 再利用（used_atあり）=> DeleteByUserID + incident
func TestAuthUsecase_Refresh_Replay_R4(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	refreshPlain := "used"
	ua := "UA"
	userID := int64(1)
	usedAt := time.Now().Add(-1 * time.Minute)

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateRefresh", mock.Anything, refreshPlain, ua).Return(nil)

	rtRepo.On("FindByHash", mock.Anything, mock.AnythingOfType("string")).Return(model.RefreshToken{
		ID:        "rt-used",
		UserID:    userID,
		UserAgent: ua,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		UsedAt:    &usedAt,
	}, true, nil)

	rtRepo.On("DeleteByUserID", mock.Anything, userID).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Refresh(ctx, refreshPlain, ua, "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrSecurityIncident)

	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② R5: user_agent違い => DeleteByUserID + incident
func TestAuthUsecase_Refresh_UserAgentMismatch_R5(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	refreshPlain := "ua-mismatch"
	ua := "UA-NEW"
	userID := int64(1)

	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)
	v.On("ValidateRefresh", mock.Anything, refreshPlain, ua).Return(nil)

	rtRepo.On("FindByHash", mock.Anything, mock.AnythingOfType("string")).Return(model.RefreshToken{
		ID:        "rt-ua",
		UserID:    userID,
		UserAgent: "UA-OLD",
		ExpiresAt: time.Now().Add(10 * time.Minute),
		UsedAt:    nil,
	}, true, nil)

	rtRepo.On("DeleteByUserID", mock.Anything, userID).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Refresh(ctx, refreshPlain, ua, "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrSecurityIncident)

	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// =====================
// Logout（方針② logout: L1/L3）
// =====================

// 方針② L1: 正常（FindByHash -> DeleteByID）
func TestAuthUsecase_Logout_Success_L1(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	refreshPlain := "refresh-plain"

	v.On("ValidateLogout", mock.Anything).Return(nil)

	rtRepo.On("FindByHash", mock.Anything, mock.AnythingOfType("string")).Return(model.RefreshToken{
		ID:     "rt-logout",
		UserID: 1,
	}, true, nil)

	rtRepo.On("DeleteByID", mock.Anything, "rt-logout").Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Logout(ctx, refreshPlain)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "logout success", res.Message)

	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}

// 方針② L3: cookie削除など（token空）=> 401
func TestAuthUsecase_Logout_EmptyToken_L3(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	v.On("ValidateLogout", mock.Anything).Return(nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.Logout(ctx, "")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, usecase.ErrUnauthorized)

	v.AssertExpectations(t)
}

// =====================
// ForceLogout（方針② 強制ログアウト: F1/F2）
// =====================

func TestAuthUsecase_ForceLogout_Success_F1F2(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	rtRepo := new(MockRefreshTokenRepository)
	v := new(MockAuthValidator)

	targetUserID := int64(10)

	v.On("ValidateForceLogout", mock.Anything, targetUserID).Return(nil)

	userRepo.On("IncrementTokenVersion", mock.Anything, targetUserID).Return(nil)
	rtRepo.On("DeleteByUserID", mock.Anything, targetUserID).Return(nil)

	// 更新後取得して new_token_version を返す
	userRepo.On("FindByID", mock.Anything, targetUserID).Return(&model.User{
		ID:           targetUserID,
		Email:        "user@test.com",
		Role:         model.RoleUser,
		TokenVersion: 5,
		IsActive:     true,
	}, nil)

	u := newAuthUC(userRepo, rtRepo, v)

	res, err := u.ForceLogout(ctx, targetUserID)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, targetUserID, res.UserID)
	assert.Equal(t, 5, res.NewTokenVersion)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	v.AssertExpectations(t)
}
