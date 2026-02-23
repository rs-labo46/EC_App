package unit

import (
	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/usecase" // ★ 追加：usecaseパッケージを参照する
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// =====================
// モック: UserRepository
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
// モック: AuthValidator
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
// モック: RefreshTokenRepository
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
	args := m.Called(ctx, mock.AnythingOfType("time.Time"))
	return args.Get(0).(int64), args.Error(1)
}

// =====================
// helper: bcrypt
// =====================

func mustHash(t *testing.T, plain string) string {
	t.Helper()

	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt failed: %v", err)
	}
	return string(b)
}

// =====================
// Test: Register（方針② login周辺の前提作りにもなる）
// =====================

func TestAuthUsecase_Register_Success(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	validator := new(MockAuthValidator)
	rtRepo := new(MockRefreshTokenRepository) // Registerでは使わないがDI上必要

	email := "user@test.com"
	pass := "CorrectPW"

	validator.On("ValidateRegister", mock.Anything, email, pass).Return(nil)

	// Createに渡されるuserの中身も最低限検証できる
	userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *model.User) bool {
		// 小学生向けに言うと：ここで「保存されるユーザーの形が正しいか」を見てる
		return u.Email == email && u.IsActive == true && u.TokenVersion == 0 && u.PasswordHash != ""
	})).Return(nil)

	u := usecase.NewAuthUsecase(config.Config{JWTSecret: "test-secret"}, userRepo, rtRepo, validator)

	resp, err := u.Register(ctx, usecase.AuthRegisterRequest{Email: email, Password: pass})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, email, resp.User.Email)

	userRepo.AssertExpectations(t)
	validator.AssertExpectations(t)
}

// =====================
// Test: Login 正常（方針② login A1）
// =====================

func TestAuthUsecase_Login_Success_A1(t *testing.T) {
	ctx := context.Background()

	userRepo := new(MockUserRepository)
	validator := new(MockAuthValidator)
	rtRepo := new(MockRefreshTokenRepository)

	email := "user@test.com"
	pass := "CorrectPW"

	// 期限切れ掃除は失敗してもログイン継続なので、呼ばれたら0件でOKにする
	rtRepo.On("DeleteExpired", mock.Anything, mock.AnythingOfType("time.Time")).Return(int64(0), nil)

	validator.On("ValidateLogin", mock.Anything, email, pass).Return(nil)

	userRepo.On("FindByEmail", mock.Anything, email).Return(&model.User{
		ID:           1,
		Email:        email,
		PasswordHash: mustHash(t, pass),
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     true,
	}, nil)

	// last_login更新は失敗しても継続なので、呼ばれてもOK
	userRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	// refresh保存が呼ばれる（中身はランダムなので型だけ見る）
	rtRepo.On("Create", mock.Anything, mock.AnythingOfType("model.RefreshToken")).Return(nil)

	u := usecase.NewAuthUsecase(config.Config{JWTSecret: "test-secret"}, userRepo, rtRepo, validator)

	res, err := u.Login(ctx, usecase.AuthLoginRequest{Email: email, Password: pass}, "UserAgent", "127.0.0.1")

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, res.Body.Token.AccessToken) // 具体値はテストで固定できないので「空じゃない」を見る
	assert.Greater(t, res.Body.Token.ExpiresIn, 0)
	assert.NotEmpty(t, res.RefreshTokenPlain)
	assert.NotEmpty(t, res.CsrfTokenPlain)

	userRepo.AssertExpectations(t)
	rtRepo.AssertExpectations(t)
	validator.AssertExpectations(t)
}
