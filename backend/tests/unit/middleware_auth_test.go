package unit

import (
	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/middleware"
	"app/internal/repository"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// =====================
// レスポンス確認用（any禁止）
// =====================

type mwErrorResponse struct {
	Error string `json:"error"`
}

type mwOKResponse struct {
	UserID       int64  `json:"user_id"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
}

// =====================
// UserRepository モック（middleware専用：名前衝突回避）
// =====================

type MockUserRepoForMiddleware struct {
	mock.Mock
}

func (m *MockUserRepoForMiddleware) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepoForMiddleware) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	u, _ := args.Get(0).(*model.User)
	return u, args.Error(1)
}

func (m *MockUserRepoForMiddleware) FindByID(ctx context.Context, id int64) (*model.User, error) {
	args := m.Called(ctx, id)
	u, _ := args.Get(0).(*model.User)
	return u, args.Error(1)
}

func (m *MockUserRepoForMiddleware) Update(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepoForMiddleware) IncrementTokenVersion(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

var _ repository.UserRepository = (*MockUserRepoForMiddleware)(nil)

// =====================
// helper
// =====================

func mustMakeJWT(t *testing.T, secret string, sub int64, role string, tv int, signingMethod jwt.SigningMethod) string {
	t.Helper()

	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"tv":   tv,
		"iat":  1,
		"exp":  9999999999,
	}

	token := jwt.NewWithClaims(signingMethod, claims)

	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString failed: %v", err)
	}
	return s
}

func runRequest(t *testing.T, e *echo.Echo, method string, path string, authHeader string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeMWError(t *testing.T, rec *httptest.ResponseRecorder) mwErrorResponse {
	t.Helper()
	var r mwErrorResponse
	_ = json.NewDecoder(rec.Body).Decode(&r)
	return r
}

func decodeMWOK(t *testing.T, rec *httptest.ResponseRecorder) mwOKResponse {
	t.Helper()
	var r mwOKResponse
	_ = json.NewDecoder(rec.Body).Decode(&r)
	return r
}

// =====================
// AuthJWT（SEC1/SEC2）
// =====================

// Authorizationなし => 401
func TestMiddleware_AuthJWT_Unauthorized_NoHeader(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg))

	rec := runRequest(t, e, http.MethodGet, "/protected", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)
}

// Bearer形式じゃない => 401
func TestMiddleware_AuthJWT_Unauthorized_BadScheme(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Token abc.def.ghi")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)
}

// 署名違い => 401（SEC2系）
func TestMiddleware_AuthJWT_Unauthorized_BadSignature(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "correct-secret"}

	raw := mustMakeJWT(t, "wrong-secret", 1, "USER", 0, jwt.SigningMethodHS256)

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Bearer "+raw)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)
}

// アルゴリズム違い（HS512）=> 401（SEC1/SEC2寄せ）
func TestMiddleware_AuthJWT_Unauthorized_WrongAlg(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	raw := mustMakeJWT(t, cfg.JWTSecret, 1, "USER", 0, jwt.SigningMethodHS512)

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Bearer "+raw)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)
}

// 正常：ctxに値が入る
func TestMiddleware_AuthJWT_Success_SetsContext(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	raw := mustMakeJWT(t, cfg.JWTSecret, 123, "USER", 7, jwt.SigningMethodHS256)

	e.GET("/protected", func(c echo.Context) error {
		userID, _ := c.Get(middleware.CtxUserIDKey).(int64)
		role, _ := c.Get(middleware.CtxUserRoleKey).(string)
		tv, _ := c.Get(middleware.CtxTokenVersionKey).(int)

		return c.JSON(http.StatusOK, mwOKResponse{
			UserID:       userID,
			Role:         role,
			TokenVersion: tv,
		})
	}, middleware.AuthJWT(cfg))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Bearer "+raw)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := decodeMWOK(t, rec)
	assert.Equal(t, int64(123), body.UserID)
	assert.Equal(t, "USER", body.Role)
	assert.Equal(t, 7, body.TokenVersion)
}

// =====================
// TokenVersionGuard（A4/F3）
// =====================

// AuthJWT無しでGuardだけ => 401
func TestMiddleware_TokenVersionGuard_Unauthorized_MissingContext(t *testing.T) {
	e := echo.New()
	userRepo := new(MockUserRepoForMiddleware)

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.TokenVersionGuard(userRepo))

	rec := runRequest(t, e, http.MethodGet, "/protected", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)
}

// tv不一致 => 401（A4/F3）
func TestMiddleware_TokenVersionGuard_Unauthorized_TokenVersionMismatch(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	userRepo := new(MockUserRepoForMiddleware)

	raw := mustMakeJWT(t, cfg.JWTSecret, 1, "USER", 0, jwt.SigningMethodHS256)

	userRepo.On("FindByID", mock.Anything, int64(1)).Return(&model.User{
		ID:           1,
		Email:        "user@test.com",
		Role:         model.RoleUser,
		TokenVersion: 1, // 不一致
		IsActive:     true,
	}, nil)

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg), middleware.TokenVersionGuard(userRepo))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Bearer "+raw)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	body := decodeMWError(t, rec)
	assert.Equal(t, "unauthorized", body.Error)

	userRepo.AssertExpectations(t)
}

// tv一致 => 200
func TestMiddleware_TokenVersionGuard_Success(t *testing.T) {
	e := echo.New()
	cfg := config.Config{JWTSecret: "test-secret"}

	userRepo := new(MockUserRepoForMiddleware)

	raw := mustMakeJWT(t, cfg.JWTSecret, 1, "USER", 5, jwt.SigningMethodHS256)

	userRepo.On("FindByID", mock.Anything, int64(1)).Return(&model.User{
		ID:           1,
		Email:        "user@test.com",
		Role:         model.RoleUser,
		TokenVersion: 5, // 一致
		IsActive:     true,
	}, nil)

	e.GET("/protected", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	}, middleware.AuthJWT(cfg), middleware.TokenVersionGuard(userRepo))

	rec := runRequest(t, e, http.MethodGet, "/protected", "Bearer "+raw)
	assert.Equal(t, http.StatusOK, rec.Code)

	userRepo.AssertExpectations(t)
}
