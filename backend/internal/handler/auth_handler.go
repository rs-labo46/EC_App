package handler

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"time"

	auth "app/internal/usecase/auth_usecase"
)

type AuthHandler struct {
	registerUC   *auth.RegisterUserUsecase // 会員登録usecase
	loginUC      *auth.LoginUsecase        // ログインusecase
	refreshTTL   time.Duration             // refresh/csrf cookie の有効期限
	cookieSecure bool
}

// DIコンストラクタ
func NewAuthHandler(
	registerUC *auth.RegisterUserUsecase,
	loginUC *auth.LoginUsecase,
	refreshTTL time.Duration,
) *AuthHandler {
	return &AuthHandler{
		registerUC:   registerUC,
		loginUC:      loginUC,
		refreshTTL:   refreshTTL,
		cookieSecure: envBool("COOKIE_SECURE", true),
	}
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "TRUE", "True":
		return true
	case "0", "false", "FALSE", "False":
		return false
	default:
		return def
	}
}

// /auth/register のリクエストボディ。
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// /auth/login のリクエストボディ。
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// RegisterはPOST /auth/registerのハンドラ
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "METHOD_NOT_ALLOWED"})
		return
	}

	var req registerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "VALIDATION_ERROR"})
		return
	}

	out, err := h.registerUC.Execute(r.Context(), auth.RegisterUserInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidEmailFormat, auth.ErrPasswordTooShort, auth.ErrWeakPassword:
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "VALIDATION_ERROR"})
			return
		case auth.ErrEmailAlreadyExists:
			writeJSON(w, http.StatusConflict, errorResponse{Error: "CONFLICT"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "INTERNAL"})
			return
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// LoginはPOST /auth/login のハンドラ。
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "METHOD_NOT_ALLOWED"})
		return
	}

	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "VALIDATION_ERROR"})
		return
	}

	// User-Agentを取得（refreshtokenに紐付ける）
	userAgent := r.Header.Get("User-Agent")

	out, side, err := h.loginUC.Execute(r.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		UserAgent: userAgent,
	})
	if err != nil {
		switch err {
		case auth.ErrInvalidCredentials:
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "UNAUTHORIZED"})
			return
		case auth.ErrUserInactive:
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "FORBIDDEN"})
			return
		default:
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "INTERNAL"})
			return
		}
	}

	// refresh cookie
	h.setRefreshCookie(w, side.PlainRefreshToken)

	//csrf cookie
	csrfToken, genErr := generateSecureToken(32)
	if genErr != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "INTERNAL"})
		return
	}
	h.setCsrfCookie(w, csrfToken)

	//JSONレスポンス（user + token）
	writeJSON(w, http.StatusOK, out)
}

// refreshtoken をCookieにセット。
func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, plainRefresh string) {
	exp := time.Now().Add(h.refreshTTL)

	cookie := &http.Cookie{
		Name:     "refresh",
		Value:    plainRefresh,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	}
	http.SetCookie(w, cookie)
}

// csrftokenをCookieにセット
func (h *AuthHandler) setCsrfCookie(w http.ResponseWriter, csrfToken string) {
	exp := time.Now().Add(h.refreshTTL)

	cookie := &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
	}
	http.SetCookie(w, cookie)
}

// ランダム文字列を作る。
func generateSecureToken(bytesLen int) (string, error) {
	if bytesLen <= 0 {
		bytesLen = 32
	}

	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// リクエストボディのJSONを読み取り。
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// JSONレスポンスを書き込み。
func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
