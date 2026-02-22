package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func Test_UserAuth_LoginMe_ForceLogout_InvalidatesOldAccess(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()

	//管理者でログインしてaccess_tokenを得る
	loginReq := LoginRequest{Email: "a@example.com", Password: "password123"}
	loginJSON, err := json.Marshal(loginReq)
	if err != nil {
		t.Fatalf("json.Marshal(LoginRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/auth/login", "", loginJSON)
	requireStatus(t, resp, http.StatusOK, body)

	login := mustDecodeLogin(t, body)

	//返ってきたユーザーがADMINであること
	if login.User.Role != "ADMIN" {
		t.Fatalf("role must be ADMIN, got=%s", login.User.Role)
	}

	//tokenが空でないこと
	if strings.TrimSpace(login.Token.AccessToken) == "" {
		t.Fatalf("access token empty: body=%s", string(body))
	}

	oldAccess := login.Token.AccessToken
	targetUserID := login.User.ID
	oldTV := login.User.TokenVersion

	///meが200で返るか（AuthJWT + TokenVersionGuardが通るか）
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/me", oldAccess, nil)
	requireStatus(t, resp, http.StatusOK, body)

	//force-logoutを実行する（ADMINのみ）
	path := "/admin/users/" + toStr(targetUserID) + "/force-logout"
	resp, body = c.doJSON(ctx, t, http.MethodPost, path, oldAccess, nil)
	requireStatus(t, resp, http.StatusOK, body)

	fr := mustDecodeForceLogout(t, body)

	//対象user_idが一致するか
	if fr.UserID != targetUserID {
		t.Fatalf("user_id mismatch want=%d got=%d", targetUserID, fr.UserID)
	}

	//token_versionが増えていること
	if fr.NewTokenVersion <= oldTV {
		t.Fatalf("token_version should increase old=%d new=%d", oldTV, fr.NewTokenVersion)
	}

	//古いJWTは無効になって /meが401になるか
	time.Sleep(30 * time.Millisecond)

	resp, body = c.doJSON(ctx, t, http.MethodGet, "/me", oldAccess, nil)
	requireStatus(t, resp, http.StatusUnauthorized, body)

	//Error {error:string} の形で返るか
	er := mustDecodeError(t, body)
	if strings.TrimSpace(er.Error) == "" {
		t.Fatalf("error message empty: body=%s", string(body))
	}
}
