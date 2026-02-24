package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// /admin/ordersのitems内の要素。
type AdminOrderItem struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
}

// /admin/orders の配列要素
type AdminOrder struct {
	ID         int64            `json:"id"`
	UserID     int64            `json:"user_id"`
	Status     string           `json:"status"`
	TotalPrice int64            `json:"total_price"`
	CreatedAt  string           `json:"created_at"`
	Items      []AdminOrderItem `json:"items"`
}

// /admin/orders/{id}/status の入力
type OrderStatusUpdateRequest struct {
	Status string `json:"status"`
}

// /orders 作成用（bodyは address_id のみ）
type adminOrderCreateBody struct {
	AddressID int64 `json:"address_id"`
}

// /admin/orders の配列をデコード
func mustDecodeAdminOrders(t *testing.T, body []byte) []AdminOrder {
	t.Helper()

	var v []AdminOrder
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal([]AdminOrder) failed: %v body=%s", err, string(body))
	}
	return v
}

// 商品を作ってから /products?q= でIDを拾う。
func createProductAndGetID(t *testing.T, c *TestClient, ctx context.Context, access string, name string, stock int64) int64 {
	t.Helper()

	// 商品作成（admin）
	req := ProductCreateRequest{
		Name:        name,
		Description: "admin orders test",
		Price:       1000,
		Stock:       stock,
		IsActive:    true,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(ProductCreateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/admin/products", access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	// 公開一覧で検索してIDを拾う
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+name+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("product not found after create: body=%s", string(body))
	}
	return list.Items[0].ID
}

// /admin/orders/{id}/status を叩いて200を確認
func updateOrderStatus(t *testing.T, c *TestClient, ctx context.Context, access string, orderID int64, status string) {
	t.Helper()

	req := OrderStatusUpdateRequest{Status: status}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(OrderStatusUpdateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPut, "/admin/orders/"+toStr(orderID)+"/status", access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)
}

// /admin/orders/{id}/status が 400 になることを確認。
func updateOrderStatusExpect400(t *testing.T, c *TestClient, ctx context.Context, access string, orderID int64, status string, wantMsg string) {
	t.Helper()

	req := OrderStatusUpdateRequest{Status: status}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(OrderStatusUpdateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPut, "/admin/orders/"+toStr(orderID)+"/status", access, reqJSON)
	requireStatus(t, resp, http.StatusBadRequest, body)

	er := mustDecodeError(t, body)
	if er.Error != wantMsg {
		t.Fatalf("error mismatch want=%s got=%s body=%s", wantMsg, er.Error, string(body))
	}
}

// getProductStock は /products/{id} を叩いて stock を返す。
func getProductStock(t *testing.T, c *TestClient, ctx context.Context, productID int64) int64 {
	t.Helper()

	resp, body := c.doJSON(ctx, t, http.MethodGet, "/products/"+toStr(productID), "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	p := mustDecodeProduct(t, body)
	return p.Stock
}

// /addresses を作って address_id を返す（AdminOrdersテスト用）
func createAddressForAdminOrderTest(t *testing.T, c *TestClient, ctx context.Context, access string) int64 {
	t.Helper()

	req := map[string]string{
		"postal_code": "5300001",
		"prefecture":  "大阪府",
		"city":        "大阪市北区",
		"line1":       "梅田1-1-1",
		"line2":       "",
		"name":        "管理者注文テスト住所",
		"phone":       "09000000000",
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(address) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/addresses", access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)

	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("json.Unmarshal(create address resp) failed: %v body=%s", err, string(body))
	}
	if out.ID <= 0 {
		t.Fatalf("invalid address id: %d body=%s", out.ID, string(body))
	}
	return out.ID
}

// /orders を叩いて注文を作る（X-Idempotency-Key をヘッダーに付ける）
// ※ TestClient.doJSON はヘッダー追加が難しいため、ここだけ net/http で直接叩く
func placeOrderForAdminOrderTest(t *testing.T, c *TestClient, ctx context.Context, access string, addressID int64, idemKey string) Order {
	t.Helper()

	b, err := json.Marshal(adminOrderCreateBody{AddressID: addressID})
	if err != nil {
		t.Fatalf("json.Marshal(order body) failed: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/orders", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("X-Idempotency-Key", idemKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP.Do failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}

	requireStatus(t, resp, http.StatusOK, bodyBytes)
	return mustDecodeOrder(t, bodyBytes) // orders_test.go にある Order の decode を流用
}

func Test_AdminOrders_List_And_StatusUpdate_And_CancelRules(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()

	// 管理者でログインする
	access := adminLogin(t, c, ctx)

	// テスト開始前にカートを空にする（helpers_test.go）
	clearCart(t, c, ctx, access)

	// 管理者一覧が取得できること（配列が返ることだけ確認）
	resp, body := c.doJSON(ctx, t, http.MethodGet, "/admin/orders?page=1&limit=20", access, nil)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeAdminOrders(t, body)

	// ---- ここから「必ず注文を作る」：住所必須化に追従するため ----

	// 注文用住所を作る（address_id を固定にしない）
	addressID := createAddressForAdminOrderTest(t, c, ctx, access)

	// 商品を作る（stock=5）
	productName := "E2E-AdminOrder-Product-" + time.Now().Format("20060102-150405.000000000")
	productID := createProductAndGetID(t, c, ctx, access, productName, 5)

	// カートに2個入れる
	addToCart(t, c, ctx, access, productID, 2)

	// 注文を作る（PENDING想定）
	orderKey := "e2e-admin-order-key-" + time.Now().Format("150405.000000000")
	order := placeOrderForAdminOrderTest(t, c, ctx, access, addressID, orderKey)
	if order.ID <= 0 {
		t.Fatalf("order id should be > 0: order=%v", order)
	}
	orderID := order.ID

	// 注文で在庫が減っているはず（5→3）
	stockAfterOrder := getProductStock(t, c, ctx, productID)
	if stockAfterOrder != 3 {
		t.Fatalf("stock should be 3 after order, got=%d", stockAfterOrder)
	}

	// PENDING → CANCELED にして在庫が戻ること（3→5）
	updateOrderStatus(t, c, ctx, access, orderID, "CANCELED")

	stockAfterCancel := getProductStock(t, c, ctx, productID)
	if stockAfterCancel != 5 {
		t.Fatalf("stock should be restored to 5 after cancel, got=%d", stockAfterCancel)
	}

	// ---- もう1回、注文を作って SHIPPED にしてから CANCELED を試す ----
	clearCart(t, c, ctx, access)
	addToCart(t, c, ctx, access, productID, 2)

	orderKey2 := "e2e-admin-order-key-2-" + time.Now().Format("150405.000000000")
	order2 := placeOrderForAdminOrderTest(t, c, ctx, access, addressID, orderKey2)
	if order2.ID <= 0 {
		t.Fatalf("order2 id should be > 0: order=%v", order2)
	}
	orderID2 := order2.ID

	// PENDING → SHIPPED は許可されること（200）
	updateOrderStatus(t, c, ctx, access, orderID2, "SHIPPED")

	// SHIPPED → CANCELED は400になること
	updateOrderStatusExpect400(t, c, ctx, access, orderID2, "CANCELED", "cannot change shipped order")
}

// =====================
// DB helpers (audit_logs 検証用)
// =====================

func mustOpenDB(t *testing.T) *sql.DB {
	t.Helper()

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	if host == "" || port == "" || user == "" || pass == "" || dbname == "" {
		t.Fatalf("POSTGRES_* env is required for DB check (HOST/PORT/USER/PASSWORD/DB)")
	}

	// pgx stdlib DSN
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, pass, host, port, dbname)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open(pgx) failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("db.Ping failed: %v", err)
	}
	return db
}

// audit_logs に「注文ステータス更新」のログがあるか確認する
func assertAuditOrderStatus(t *testing.T, db *sql.DB, orderID int64, beforeStatus string, afterStatus string) {
	t.Helper()

	wantBefore := fmt.Sprintf(`{"status":"%s"}`, beforeStatus)
	wantAfter := fmt.Sprintf(`{"status":"%s"}`, afterStatus)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cnt int
	// テーブル/カラム名が違う場合はここで落ちる（その時はエラー文を貼ってくれれば合わせる）
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM audit_logs
		WHERE action = 'UPDATE_ORDER_STATUS'
		  AND resource_type = 'ORDER'
		  AND resource_id = $1
		  AND before_json = $2
		  AND after_json = $3
	`, orderID, wantBefore, wantAfter).Scan(&cnt)

	if err != nil {
		t.Fatalf("audit_logs query failed: %v", err)
	}
	if cnt < 1 {
		t.Fatalf("audit log not found: order_id=%d before=%s after=%s", orderID, wantBefore, wantAfter)
	}
}

// orders テーブルから現在の status を取る（before_status を作る用）
func mustGetOrderStatus(t *testing.T, db *sql.DB, orderID int64) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var st string
	err := db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&st)
	if err != nil {
		t.Fatalf("select orders.status failed: %v", err)
	}
	if st == "" {
		t.Fatalf("empty status for order_id=%d", orderID)
	}
	return st
}
