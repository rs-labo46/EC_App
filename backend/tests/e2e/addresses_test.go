package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"testing"
	"time"
)

// login のレスポンス構造。
type loginResp struct {
	User struct {
		ID int64 `json:"id"`
	} `json:"user"`
	Token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	} `json:"token"`
}

type addressDTO struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	PostalCode string `json:"postal_code"`
	Prefecture string `json:"prefecture"`
	City       string `json:"city"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	IsDefault  bool   `json:"is_default"`
}

func Test_Addresses_FullFlow_Create_List_Default_Update_Delete(t *testing.T) {
	// テスト対象APIのベースURLを環境変数から読む
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://localhost:8080"

	}

	// cookie jar付きのHTTPクライアントを作る。
	client := newHTTPClient(t)

	// テスト用のユーザーを作るため、メールアドレスをユニークにする（重複登録を避ける）。
	email := fmt.Sprintf("addr_%d@test.com", time.Now().UnixNano())

	// テスト用のパスワード
	password := "CorrectPW123!"

	//Register（ユーザー登録）
	///auth/registerが200を返し、ユーザーが作成できること。
	postJSONMapNoResp(t, client, base+"/auth/register", "", map[string]string{
		"email":    email,
		"password": password,
	}, http.StatusOK)

	//Login（ログイン）
	///auth/loginが200を返し、access_token を受け取れること。
	lr := postJSONMapLoginResp(t, client, base+"/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, http.StatusOK)

	//access_tokenが空なら、この後の/addressesが全部失敗するのでテストを止める。
	access := lr.Token.AccessToken
	if access == "" {
		t.Fatalf("access_token is empty")
	}

	//Address Create（住所作成）
	//POST /addresses が200を返し、住所IDが採番されること。
	created := postJSONMapAddressDTO(t, client, base+"/addresses", access, map[string]string{
		"postal_code": "5300001",
		"prefecture":  "大阪府",
		"city":        "大阪市北区",
		"line1":       "梅田1-1-1",
		"line2":       "テストビル101",
		"name":        "山田太郎",
		"phone":       "09000000000",
	}, http.StatusOK)

	//作成直後の住所IDが正しく入っていることを確認する。
	if created.ID <= 0 {
		t.Fatalf("created.id invalid: %d", created.ID)
	}

	//Address List（住所一覧取得）
	//GET /addressesが200を返し、作成した住所が1件だけ取得できること。
	list1 := getAddressList(t, client, base+"/addresses", access, http.StatusOK)
	if len(list1) != 1 {
		t.Fatalf("expected 1 address, got %d", len(list1))
	}
	// 取得した住所が、作成した住所IDと一致することを確認する。
	if list1[0].ID != created.ID {
		t.Fatalf("unexpected address id: got %d want %d", list1[0].ID, created.ID)
	}

	//Set Default（デフォルト住所切替)
	//POST /addresses/{id}/defaultが200を返すこと。
	postNoBodyNoResp(t, client, fmt.Sprintf("%s/addresses/%d/default", base, created.ID), access, http.StatusOK)

	//default反映確認（一覧で is_default=true を確認)
	//GET /addresses で、その住所のis_defaultがtrueになっていること。
	list2 := getAddressList(t, client, base+"/addresses", access, http.StatusOK)
	if len(list2) != 1 {
		t.Fatalf("expected 1 address, got %d", len(list2))
	}
	if !list2[0].IsDefault {
		t.Fatalf("expected is_default=true, got false")
	}

	//Update（住所更新）
	//PATCH /addresses/{id}が200を返し、住所内容が更新されること。
	patchJSONMapNoResp(t, client, fmt.Sprintf("%s/addresses/%d", base, created.ID), access, map[string]string{
		"postal_code": "5300002",
		"prefecture":  "大阪府",
		"city":        "大阪市北区",
		"line1":       "梅田2-2-2",
		"line2":       "",
		"name":        "山田太郎",
		"phone":       "09011112222",
	}, http.StatusOK)

	// 更新反映確認（一覧で値が変わっていることを確認）
	//GET /addresses で、更新したpostal_code / line1 が反映されていること。
	list3 := getAddressList(t, client, base+"/addresses", access, http.StatusOK)
	if len(list3) != 1 {
		t.Fatalf("expected 1 address, got %d", len(list3))
	}
	if list3[0].PostalCode != "5300002" || list3[0].Line1 != "梅田2-2-2" {
		t.Fatalf("update not applied: %+v", list3[0])
	}

	//Delete（住所削除
	//DELETE /addresses/{id}が200を返し、住所が削除されること。
	deleteNoResp(t, client, fmt.Sprintf("%s/addresses/%d", base, created.ID), access, http.StatusOK)

	//削除反映確認で一覧が空になることを確認
	//GET /addresses で 0件になること。
	list4 := getAddressList(t, client, base+"/addresses", access, http.StatusOK)
	if len(list4) != 0 {
		t.Fatalf("expected 0 address, got %d", len(list4))
	}
}

// cookie jar付きのHTTPクライアントを作る。
// loginなどでSet-Cookieがある場合でも、次のリクエストへcookieが引き継がれるようにする。
func newHTTPClient(t *testing.T) *http.Client {
	t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}

	return &http.Client{Jar: jar}
}

// POST（JSON）でリクエスト
// /auth/register のように「成功したかだけ」見たい時に使う。
func postJSONMapNoResp(t *testing.T, c *http.Client, url string, access string, body map[string]string, wantStatus int) {
	t.Helper()

	req := newJSONRequest(t, http.MethodPost, url, access, body)
	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}
}

// POST（JSON）でログインし、loginRespを返す。
// access_token を取得して以降のAPI呼び出しに使う。
func postJSONMapLoginResp(t *testing.T, c *http.Client, url string, body map[string]string, wantStatus int) loginResp {
	t.Helper()

	req := newJSONRequest(t, http.MethodPost, url, "", body)
	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}

	var out loginResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode loginResp: %v", err)
	}

	return out
}

// POST（JSON）で住所作成し、作成結果(addressDTO)を受け取る。
// 作成した住所IDを後続の default / update / delete で使う。
func postJSONMapAddressDTO(t *testing.T, c *http.Client, url string, access string, body map[string]string, wantStatus int) addressDTO {
	t.Helper()

	req := newJSONRequest(t, http.MethodPost, url, access, body)
	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}

	var out addressDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode addressDTO: %v", err)
	}

	return out
}

// PATCH（JSON）で住所更新
// 更新が成功したか（ステータスコード）
func patchJSONMapNoResp(t *testing.T, c *http.Client, url string, access string, body map[string]string, wantStatus int) {
	t.Helper()

	req := newJSONRequest(t, http.MethodPatch, url, access, body)
	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("PATCH %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}
}

// DELETEで住所削除
// 削除が成功したか（ステータスコード）
func deleteNoResp(t *testing.T, c *http.Client, url string, access string, wantStatus int) {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("new DELETE req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}

	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("DELETE %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}
}

// POST（ボディ無し）で default切替などを呼びたい。
func postNoBodyNoResp(t *testing.T, c *http.Client, url string, access string, wantStatus int) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("new POST req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}

	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}
}

// GETで住所一覧を取得して返す。
// 作成/更新/削除/default の反映を確認するために、一覧取得で状態を見る。
func getAddressList(t *testing.T, c *http.Client, url string, access string, wantStatus int) []addressDTO {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("new GET req: %v", err)
	}
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}

	resp := doRequest(t, c, req)
	defer resp.Body.Close()

	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s: want %d got %d", url, wantStatus, resp.StatusCode)
	}

	var out []addressDTO
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode address list: %v", err)
	}

	return out
}

// JSONリクエストを作る
// Content-Typeをjsonにし、必要ならAuthorizationヘッダーも付ける。
func newJSONRequest(t *testing.T, method string, url string, access string, body map[string]string) *http.Request {
	t.Helper()

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("new req: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}

	return req
}

// c.Doのエラーをテスト失敗とし、呼び出し側でレスポンスを検証。
func doRequest(t *testing.T, c *http.Client, req *http.Request) *http.Response {
	t.Helper()

	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("http do: %v", err)
	}

	return resp
}

// 所有チェックが効いていて、他人の住所を操作できないことを確認する。
// ユーザーAで住所作成 → ユーザーBでAの住所をdefault/update/delete しようとして403を確認。
func Test_Addresses_Ownership_Forbidden_On_OtherUsers_Address(t *testing.T) {
	base := os.Getenv("BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	client := newHTTPClient(t)

	//ユーザーAを作成してログインする
	emailA := fmt.Sprintf("addr_ownerA_%d@test.com", time.Now().UnixNano())
	passwordA := "CorrectPW123!"

	//Register（A）
	postJSONMapNoResp(t, client, base+"/auth/register", "", map[string]string{
		"email":    emailA,
		"password": passwordA,
	}, http.StatusOK)

	//Login（A）
	lrA := postJSONMapLoginResp(t, client, base+"/auth/login", map[string]string{
		"email":    emailA,
		"password": passwordA,
	}, http.StatusOK)

	accessA := lrA.Token.AccessToken
	if accessA == "" {
		t.Fatalf("accessA is empty")
	}

	//Aの住所を作成
	createdA := postJSONMapAddressDTO(t, client, base+"/addresses", accessA, map[string]string{
		"postal_code": "1000001",
		"prefecture":  "東京都",
		"city":        "千代田区",
		"line1":       "テスト町1-1-1",
		"line2":       "",
		"name":        "所有者A",
		"phone":       "09000000001",
	}, http.StatusOK)

	if createdA.ID <= 0 {
		t.Fatalf("createdA.id invalid: %d", createdA.ID)
	}

	//ユーザーBを作成してログインする
	emailB := fmt.Sprintf("addr_ownerB_%d@test.com", time.Now().UnixNano())
	passwordB := "CorrectPW123!"

	//Register（B）
	postJSONMapNoResp(t, client, base+"/auth/register", "", map[string]string{
		"email":    emailB,
		"password": passwordB,
	}, http.StatusOK)

	//Login（B）
	lrB := postJSONMapLoginResp(t, client, base+"/auth/login", map[string]string{
		"email":    emailB,
		"password": passwordB,
	}, http.StatusOK)

	accessB := lrB.Token.AccessToken
	if accessB == "" {
		t.Fatalf("accessB is empty")
	}

	//BがAの住所をdefaultにしようとして403を確認
	postNoBodyNoResp(t, client, fmt.Sprintf("%s/addresses/%d/default", base, createdA.ID), accessB, http.StatusForbidden)

	//BがAの住所をupdateしようとして403を確認
	patchJSONMapNoResp(t, client, fmt.Sprintf("%s/addresses/%d", base, createdA.ID), accessB, map[string]string{
		"postal_code": "9999999",
		"prefecture":  "不正県",
		"city":        "不正市",
		"line1":       "不正1-1-1",
		"line2":       "",
		"name":        "不正更新",
		"phone":       "09099999999",
	}, http.StatusForbidden)

	//BがAの住所をdeleteしようとして403を確認
	deleteNoResp(t, client, fmt.Sprintf("%s/addresses/%d", base, createdA.ID), accessB, http.StatusForbidden)

	//Aで削除しておく
	deleteNoResp(t, client, fmt.Sprintf("%s/addresses/%d", base, createdA.ID), accessA, http.StatusOK)
}
