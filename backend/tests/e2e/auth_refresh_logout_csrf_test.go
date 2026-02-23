package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenVersion int64  `json:"token_version"`
}

func mustReadAllBytes(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll 失敗しました: %v", err)
	}
	return b
}

func requireStatusOneOf(t *testing.T, resp *http.Response, body []byte, wants ...int) {
	t.Helper()
	for _, w := range wants {
		if resp.StatusCode == w {
			return
		}
	}
	t.Fatalf("status=%d want one of=%v body=%s", resp.StatusCode, wants, string(body))
}

func mustDecodeRefresh(t *testing.T, body []byte) refreshResp {
	t.Helper()
	var v refreshResp
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(refreshResp) 失敗: %v body=%s", err, string(body))
	}
	return v
}

func getCookieValueFromJar(t *testing.T, c *TestClient, rawURL string, name string) string {
	t.Helper()

	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse 失敗: %v", err)
	}

	cookies := c.HTTP.Jar.Cookies(u)
	for _, ck := range cookies {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

// refresh を叩く（CSRF Double Submit：cookie csrf_token と header X-CSRF-Token 同じ値）
func callRefresh(t *testing.T, c *TestClient, ctx context.Context, csrfToken string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/auth/refresh", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("NewRequest refresh 失敗: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)

	//もしバック側で Origin チェックを入れている場合に備えて付ける
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("HTTP.Do refresh 失敗: %v", err)
	}
	defer resp.Body.Close()
	body := mustReadAllBytes(t, resp)
	return resp, body
}

// Cookie を明示的に固定して refresh を叩く
func callRefreshWithFixedCookies(t *testing.T, c *TestClient, ctx context.Context, csrfToken string, refreshCookie string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/auth/refresh", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("NewRequest refresh fixed 失敗: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Origin", "http://localhost:3000")

	//jar の自動cookie付与を避けるため、Cookieヘッダを自前でセット
	//csrf_token は double submit の cookie側 として必要なので一緒に送る
	req.Header.Set("Cookie", "refresh_token="+refreshCookie+"; csrf_token="+csrfToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP.Do refresh fixed 失敗: %v", err)
	}
	defer resp.Body.Close()
	body := mustReadAllBytes(t, resp)
	return resp, body
}

// logout を叩く（CSRF必須 + bearer必須）
func callLogout(t *testing.T, c *TestClient, ctx context.Context, access string, csrfToken string, withCsrf bool) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/auth/logout", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("NewRequest logout 失敗: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("Origin", "http://localhost:3000")

	if withCsrf {
		req.Header.Set("X-CSRF-Token", csrfToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		t.Fatalf("HTTP.Do logout 失敗: %v", err)
	}
	defer resp.Body.Close()
	body := mustReadAllBytes(t, resp)
	return resp, body
}

// refresh 正常 + rotation + replay（古いrefreshを再利用）で失敗
func Test_Auth_Refresh_Rotation_And_ReplayDetected(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()

	//ユーザーをユニークに作ってlogin
	email := "e2e_refresh_" + time.Now().Format("20060102_150405.000000000") + "@test.com"
	pass := "CorrectPW123!"

	//register
	reg := registerRequest{Email: email, Password: pass}
	regJSON, _ := json.Marshal(reg)
	resp, body := c.doJSON(ctx, t, http.MethodPost, "/auth/register", "", regJSON)
	requireStatus(t, resp, http.StatusOK, body)

	//login（cookie jar に refresh_token / csrf_token が入る想定）
	loginReq := LoginRequest{Email: email, Password: pass}
	loginJSON, _ := json.Marshal(loginReq)
	resp, body = c.doJSON(ctx, t, http.MethodPost, "/auth/login", "", loginJSON)
	requireStatus(t, resp, http.StatusOK, body)
	login := mustDecodeLogin(t, body)
	if strings.TrimSpace(login.Token.AccessToken) == "" {
		t.Fatalf("access token empty: body=%s", string(body))
	}

	//csrf_token cookie を取得
	csrf := getCookieValueFromJar(t, c, c.BaseURL, "csrf_token")
	if csrf == "" {
		t.Fatalf("csrf_token cookie not found (host mismatch? BASE_URL=%s)", c.BaseURL)
	}

	//refresh_token cookieを古い値として保存（rotation/replay）
	oldRefresh := getCookieValueFromJar(t, c, c.BaseURL, "refresh_token")
	if oldRefresh == "" {
		t.Fatalf("refresh_token cookie not found")
	}

	//1回目 refresh（正常）→ 新しいaccess返る + refresh cookie がローテーション
	resp, body = callRefresh(t, c, ctx, csrf)
	requireStatus(t, resp, http.StatusOK, body)
	r1 := mustDecodeRefresh(t, body)
	if strings.TrimSpace(r1.AccessToken) == "" {
		t.Fatalf("refresh returned empty access_token: body=%s", string(body))
	}

	//jar 上の refresh_token が変わっていること（rotation）
	newRefresh := getCookieValueFromJar(t, c, c.BaseURL, "refresh_token")
	if newRefresh == "" {
		t.Fatalf("refresh_token cookie missing after refresh")
	}
	if newRefresh == oldRefresh {
		t.Fatalf("refresh token should rotate (same value). old=%s new=%s", oldRefresh, newRefresh)
	}

	//古いrefresh_token をもう一度使って refresh（replay）→ 401 になる想定
	resp, body = callRefreshWithFixedCookies(t, c, ctx, csrf, oldRefresh)
	//ここは「失敗であること」を優先
	requireStatusOneOf(t, resp, body, http.StatusUnauthorized, http.StatusBadRequest)

	//Error {error:string} で返ってくること
	_ = mustDecodeError(t, body)
}

// logout 正常 + logout後refreshできない + CSRF無しlogoutは失敗
func Test_Auth_Logout_CsrfRequired_And_RefreshFailsAfterLogout(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()

	//ユーザー作成 → login
	email := "e2e_logout_" + time.Now().Format("20060102_150405.000000000") + "@test.com"
	pass := "CorrectPW123!"

	reg := registerRequest{Email: email, Password: pass}
	regJSON, _ := json.Marshal(reg)
	resp, body := c.doJSON(ctx, t, http.MethodPost, "/auth/register", "", regJSON)
	requireStatus(t, resp, http.StatusOK, body)

	loginReq := LoginRequest{Email: email, Password: pass}
	loginJSON, _ := json.Marshal(loginReq)
	resp, body = c.doJSON(ctx, t, http.MethodPost, "/auth/login", "", loginJSON)
	requireStatus(t, resp, http.StatusOK, body)
	login := mustDecodeLogin(t, body)

	access := login.Token.AccessToken
	if strings.TrimSpace(access) == "" {
		t.Fatalf("アクセストークンが空です: body=%s", string(body))
	}

	csrf := getCookieValueFromJar(t, c, c.BaseURL, "csrf_token")
	if csrf == "" {
		t.Fatalf("csrf_tokenクッキーが見つかりません BASE_URL=%s)", c.BaseURL)
	}

	//CSRF無し logoutは失敗する
	resp, body = callLogout(t, c, ctx, access, csrf, false)
	requireStatusOneOf(t, resp, body, http.StatusBadRequest, http.StatusUnauthorized)
	_ = mustDecodeError(t, body)

	//CSRFありlogoutは成功
	resp, body = callLogout(t, c, ctx, access, csrf, true)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//logout後、refreshは失敗（cookieが消されている想定）
	resp, body = callRefresh(t, c, ctx, csrf)
	requireStatusOneOf(t, resp, body, http.StatusUnauthorized, http.StatusBadRequest)
	_ = mustDecodeError(t, body)
}
