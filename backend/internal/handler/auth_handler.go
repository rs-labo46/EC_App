package handler

import (
	"errors"
	"net/http"
	"time"

	"app/internal/config"
	"app/internal/middleware"
	"app/internal/repository"
	"app/internal/usecase"
	"app/internal/validator"

	"github.com/labstack/echo/v4"
)

// Cookie名
const (
	cookieRefreshToken = "refresh_token"
	cookieCsrfToken    = "csrf_token"
	headerCsrfToken    = "X-CSRF-Token"
)

// /authのHTTPハンドラ
type AuthHandler struct {
	cfg      config.Config
	uc       *usecase.AuthUsecase
	userRepo repository.UserRepository
}

// DI
func NewAuthHandler(cfg config.Config, uc *usecase.AuthUsecase, userRepo repository.UserRepository) *AuthHandler {
	return &AuthHandler{cfg: cfg, uc: uc, userRepo: userRepo}
}
func (h *AuthHandler) Me(c echo.Context) error {
	raw := c.Get(middleware.CtxUserIDKey)
	userID, ok := raw.(int64)
	if !ok || userID <= 0 {
		return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
	}

	dto, err := h.uc.Me(c.Request().Context(), userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, dto)
}

// ルーティング登録
func (h *AuthHandler) RegisterRoutes(e *echo.Echo) {
	auth := e.Group("/auth")

	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
	auth.POST("/refresh", h.Refresh)

	auth.POST(
		"/logout",
		h.Logout,
		middleware.AuthJWT(h.cfg),
		middleware.TokenVersionGuard(h.userRepo),
	)
	// ★ /me は bearerAuth + token_version一致
	e.GET("/me", h.Me,
		middleware.AuthJWT(h.cfg),
		middleware.TokenVersionGuard(h.userRepo),
	)
}

// POST /auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req usecase.AuthRegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorJSON("invalid json"))
	}

	res, err := h.uc.Register(c.Request().Context(), req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

// POST /auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req usecase.AuthLoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorJSON("invalid json"))
	}

	ua := c.Request().UserAgent()
	ip := c.RealIP()

	result, err := h.uc.Login(c.Request().Context(), req, ua, ip)
	if err != nil {
		return h.handleError(c, err)
	}

	//refresh_tokenをHttpOnly cookieにセット
	h.setRefreshCookie(c, result.RefreshTokenPlain)

	//csrf_tokenをcookie にセット
	h.setCsrfCookie(c, result.CsrfTokenPlain)
	return c.JSON(http.StatusOK, result.Body)
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
	//CSRF検証
	if err := h.verifyDoubleSubmitCsrf(c); err != nil {
		return c.JSON(http.StatusUnauthorized, errorJSON("csrf invalid"))
	}

	//refresh_token cookieを取得
	refreshPlain, err := getCookieValue(c, cookieRefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errorJSON("refresh missing"))
	}

	ua := c.Request().UserAgent()
	ip := c.RealIP()

	result, uerr := h.uc.Refresh(c.Request().Context(), refreshPlain, ua, ip)
	if uerr != nil {
		return h.handleError(c, uerr)
	}

	//cookieを再セット
	h.setRefreshCookie(c, result.RefreshTokenPlain)
	h.setCsrfCookie(c, result.CsrfTokenPlain)
	return c.JSON(http.StatusOK, result.Body)
}

// POST /auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	// CSRF検証（Double Submit Cookie）
	if err := h.verifyDoubleSubmitCsrf(c); err != nil {
		return c.JSON(http.StatusUnauthorized, errorJSON("csrf invalid"))
	}

	//refresh_tokencookieを取得
	refreshPlain, err := getCookieValue(c, cookieRefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, errorJSON("refresh missing"))
	}

	res, uerr := h.uc.Logout(c.Request().Context(), refreshPlain)
	if uerr != nil {
		return h.handleError(c, uerr)
	}

	//cookie削除
	h.clearCookie(c, cookieRefreshToken)
	h.clearCookie(c, cookieCsrfToken)

	return c.JSON(http.StatusOK, res)
}

// helper: CSRF Double Submit Cookie 検証
func (h *AuthHandler) verifyDoubleSubmitCsrf(c echo.Context) error {
	header := c.Request().Header.Get(headerCsrfToken)
	if header == "" {
		return errors.New("csrf header missing")
	}

	cookieVal, err := getCookieValue(c, cookieCsrfToken)
	if err != nil {
		return errors.New("csrf cookie missing")
	}

	if header != cookieVal {
		return errors.New("csrf mismatch")
	}

	return nil
}

func (h *AuthHandler) handleError(c echo.Context, err error) error {
	// validator層のエラー
	switch {
	case errors.Is(err, validator.ErrInvalidInput):
		return c.JSON(http.StatusBadRequest, errorJSON(err.Error()))
	case errors.Is(err, validator.ErrEmailAlreadyUsed):
		return c.JSON(http.StatusConflict, errorJSON(err.Error()))
	case errors.Is(err, validator.ErrInvalidRefresh):
		return c.JSON(http.StatusUnauthorized, errorJSON(err.Error()))
	}

	// usecase層のエラー
	switch {
	case errors.Is(err, usecase.ErrValidation):
		return c.JSON(http.StatusBadRequest, errorJSON(err.Error()))
	case errors.Is(err, usecase.ErrConflict):
		return c.JSON(http.StatusConflict, errorJSON(err.Error()))
	case errors.Is(err, usecase.ErrUnauthorized):
		return c.JSON(http.StatusUnauthorized, errorJSON(err.Error()))
	case errors.Is(err, usecase.ErrForbidden):
		return c.JSON(http.StatusForbidden, errorJSON(err.Error()))
	case errors.Is(err, usecase.ErrSecurityIncident):
		return c.JSON(http.StatusUnauthorized, errorJSON(err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errorJSON("internal error"))
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func errorJSON(msg string) errorResponse {
	return errorResponse{Error: msg}
}

// Cookie操作
func (h *AuthHandler) setRefreshCookie(c echo.Context, value string) {
	c.SetCookie(&http.Cookie{
		Name:     cookieRefreshToken,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func (h *AuthHandler) setCsrfCookie(c echo.Context, value string) {
	c.SetCookie(&http.Cookie{
		Name:     cookieCsrfToken,
		Value:    value,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
}

func (h *AuthHandler) clearCookie(c echo.Context, name string) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: name == cookieRefreshToken,
		Secure:   h.isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
	})
}

func (h *AuthHandler) isSecureCookie() bool {
	return h.cfg.GoEnv == "prod"
}

func getCookieValue(c echo.Context, name string) (string, error) {
	ck, err := c.Cookie(name)
	if err != nil {
		return "", err
	}
	if ck.Value == "" {
		return "", errors.New("cookie empty")
	}
	return ck.Value, nil
}
