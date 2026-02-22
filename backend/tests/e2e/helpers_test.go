package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

type TestClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewTestClient(t *testing.T) *TestClient {
	t.Helper()

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New failed: %v", err)
	}

	return &TestClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{
			Jar:     jar,
			Timeout: 10 * time.Second,
		},
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type UserDTO struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"`
	IsActive     bool   `json:"is_active"`
}

type JwtAccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenVersion int64  `json:"token_version"`
}

type AuthLoginResponse struct {
	User  UserDTO        `json:"user"`
	Token JwtAccessToken `json:"token"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForceLogoutResponse struct {
	UserID          int64 `json:"user_id"`
	NewTokenVersion int64 `json:"new_token_version"`
}

func (c *TestClient) doJSON(
	ctx context.Context,
	t *testing.T,
	method string,
	path string,
	bearer string,
	bodyBytes []byte,
) (*http.Response, []byte) {
	t.Helper()

	var reqBody io.Reader
	if bodyBytes != nil {
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("HTTP.Do failed: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	return resp, data
}

func requireStatus(t *testing.T, resp *http.Response, want int, body []byte) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status=%d want=%d body=%s", resp.StatusCode, want, string(body))
	}
}

func mustDecodeError(t *testing.T, body []byte) ErrorResponse {
	t.Helper()
	var v ErrorResponse
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(ErrorResponse) failed: %v body=%s", err, string(body))
	}
	return v
}

func mustDecodeLogin(t *testing.T, body []byte) AuthLoginResponse {
	t.Helper()
	var v AuthLoginResponse
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(AuthLoginResponse) failed: %v body=%s", err, string(body))
	}
	return v
}

func mustDecodeForceLogout(t *testing.T, body []byte) ForceLogoutResponse {
	t.Helper()
	var v ForceLogoutResponse
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(ForceLogoutResponse) failed: %v body=%s", err, string(body))
	}
	return v
}

func mustDecodeSuccess(t *testing.T, body []byte) SuccessResponse {
	t.Helper()
	var v SuccessResponse
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(SuccessResponse) failed: %v body=%s", err, string(body))
	}
	return v
}

func toStr(v int64) string {
	return strconv.FormatInt(v, 10)
}

func adminLogin(t *testing.T, c *TestClient, ctx context.Context) string {
	t.Helper()

	//管理者でログインしてaccess_tokenを取得
	req := LoginRequest{Email: "a@example.com", Password: "password123"}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(LoginRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/auth/login", "", b)

	//200OKであることを確認
	requireStatus(t, resp, http.StatusOK, body)

	//JSONを構造体に変換し、tokenが空じゃないことを確認
	login := mustDecodeLogin(t, body)
	if strings.TrimSpace(login.Token.AccessToken) == "" {
		t.Fatalf("access token is empty: body=%s", string(body))
	}

	return login.Token.AccessToken
}
