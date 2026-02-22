package e2e

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type OrderItem struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
}

type Order struct {
	ID         int64       `json:"id"`
	UserID     int64       `json:"user_id"`
	Status     string      `json:"status"`
	TotalPrice int64       `json:"total_price"`
	CreatedAt  string      `json:"created_at"`
	Items      []OrderItem `json:"items"`
}

type OrderCreateRequest struct {
	AddressID int64 `json:"address_id"`
}

// Orderをデコード。
func mustDecodeOrder(t *testing.T, body []byte) Order {
	t.Helper()
	var v Order
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(Order) failed: %v body=%s", err, string(body))
	}
	return v
}

// []Orderをデコード。配列として読む。
func mustDecodeOrders(t *testing.T, body []byte) []Order {
	t.Helper()
	var v []Order
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal([]Order) failed: %v body=%s", err, string(body))
	}
	return v
}

// 公開商品を作成し、product_id を返す。一覧検索（q）でIDを拾う
func createPublicProductForOrder(t *testing.T, c *TestClient, ctx context.Context, access string, name string, stock int64) int64 {
	t.Helper()

	//商品を作成する（admin）
	create := ProductCreateRequest{
		Name:        name,
		Description: "for orders test",
		Price:       1000,
		Stock:       stock,
		IsActive:    true,
	}
	createJSON, err := json.Marshal(create)
	if err != nil {
		t.Fatalf("json.Marshal(ProductCreateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/admin/products", access, createJSON)

	//200OKを確認する
	requireStatus(t, resp, http.StatusOK, body)

	//Success を確認（messageがあること）
	_ = mustDecodeSuccess(t, body)

	//公開一覧で検索して、作った商品が見つかることを確認
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+name+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("created product not found in list: body=%s", string(body))
	}

	//最初の要素をproduct_id として使う
	return list.Items[0].ID
}

// addToCartは /cart に商品を追加。
// cart側で在庫超過を検知のため、ここが400になることもある。
func addToCart(t *testing.T, c *TestClient, ctx context.Context, access string, productID int64, qty int64) CartResponse {
	t.Helper()

	req := AddCartRequest{ProductID: productID, Quantity: qty}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(AddCartRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/cart", access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)

	return mustDecodeCart(t, body)
}

// /cart の現在状態を取得
func getCart(t *testing.T, c *TestClient, ctx context.Context, access string) CartResponse {
	t.Helper()

	resp, body := c.doJSON(ctx, t, http.MethodGet, "/cart", access, nil)
	requireStatus(t, resp, http.StatusOK, body)
	return mustDecodeCart(t, body)
}
func mustReadAll(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll failed: %v", err)
	}
	return b
}

// /orders を叩いて注文を確定（X-Idempotency-Key をヘッダーに付ける）
// 成功なら Order を返す
func placeOrder(t *testing.T, c *TestClient, ctx context.Context, access string, addressID int64, key string) Order {
	t.Helper()

	// bodyは address_id のみ
	reqBody := OrderCreateRequest{AddressID: addressID}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("json.Marshal(OrderCreateRequest) failed: %v", err)
	}

	// BaseURL は TestClient が持っているフィールドを使う
	url := c.BaseURL + "/orders"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqJSON)))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	// JSON送信
	httpReq.Header.Set("Content-Type", "application/json")

	// bearerAuth
	httpReq.Header.Set("Authorization", "Bearer "+access)

	// 二重送信防止キー（仕様書どおりヘッダー）
	httpReq.Header.Set("X-Idempotency-Key", key)

	//標準の http.Client で叩く
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP.Do failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes := mustReadAll(t, resp)

	requireStatus(t, resp, http.StatusOK, bodyBytes)
	return mustDecodeOrder(t, bodyBytes)
}

// /orders を取得
func listOrders(t *testing.T, c *TestClient, ctx context.Context, access string) []Order {
	t.Helper()

	resp, body := c.doJSON(ctx, t, http.MethodGet, "/orders", access, nil)
	requireStatus(t, resp, http.StatusOK, body)
	return mustDecodeOrders(t, body)
}

// /orders/{id} を取得
func getOrderDetail(t *testing.T, c *TestClient, ctx context.Context, access string, orderID int64) Order {
	t.Helper()

	resp, body := c.doJSON(ctx, t, http.MethodGet, "/orders/"+toStr(orderID), access, nil)
	requireStatus(t, resp, http.StatusOK, body)
	return mustDecodeOrder(t, body)
}

// /admin/inventory/{product_id} をたたく
// 同時購入を疑似的に
func updateInventory(t *testing.T, c *TestClient, ctx context.Context, access string, productID int64, newStock int64, reason string) {
	t.Helper()

	req := InventoryUpdateRequest{Stock: newStock, Reason: reason}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal(InventoryUpdateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPut, "/admin/inventory/"+toStr(productID), access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)
}

// /products/{id} を取得。在庫が減っていることを確認する
func getProduct(t *testing.T, c *TestClient, ctx context.Context, productID int64) Product {
	t.Helper()

	resp, body := c.doJSON(ctx, t, http.MethodGet, "/products/"+toStr(productID), "", nil)
	requireStatus(t, resp, http.StatusOK, body)
	return mustDecodeProduct(t, body)
}

// /orders が 400 になることを確認。error メッセージも確認。
func placeOrderExpect400(t *testing.T, c *TestClient, ctx context.Context, access string, addressID int64, key string, wantMsg string) {
	t.Helper()

	reqBody := OrderCreateRequest{AddressID: addressID}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("json.Marshal(OrderCreateRequest) failed: %v", err)
	}

	url := c.BaseURL + "/orders"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqJSON)))
	if err != nil {
		t.Fatalf("http.NewRequest failed: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+access)
	httpReq.Header.Set("X-Idempotency-Key", key)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("HTTP.Do failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes := mustReadAll(t, resp)

	requireStatus(t, resp, http.StatusBadRequest, bodyBytes)

	er := mustDecodeError(t, bodyBytes)
	if er.Error != wantMsg {
		t.Fatalf("error mismatch want=%s got=%s body=%s", wantMsg, er.Error, string(bodyBytes))
	}
}

// /addressesを作ってaddress_id を返す
func createAddressForOrder(t *testing.T, c *TestClient, ctx context.Context, access string) int64 {
	t.Helper()

	req := map[string]string{
		"postal_code": "5300001",
		"prefecture":  "大阪府",
		"city":        "大阪市北区",
		"line1":       "梅田1-1-1",
		"line2":       "",
		"name":        "注文テスト住所",
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

func Test_Orders_FullFlow_Idempotency_StockDecrease_CartCleared(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()
	//管理者でログインしてaccess_tokenを得る
	access := adminLogin(t, c, ctx)

	//公開商品を作成（stock=5）ユニーク名にして、一覧検索でIDを拾えるようにする
	name := "E2E-OrderBeans-" + time.Now().Format("20060102-150405.000000000")
	productID := createPublicProductForOrder(t, c, ctx, access, name, 5)

	//カートが空であることを先に確認する
	cart := getCart(t, c, ctx, access)
	if len(cart.Items) != 0 || cart.Total != 0 {
		t.Fatalf("cart should be empty at start: body=%v", cart)
	}

	//カートにquantity=2を入れる
	cart = addToCart(t, c, ctx, access, productID, 2)

	//カートに1件入っていること
	if len(cart.Items) != 1 {
		t.Fatalf("cart should have 1 item after add: cart=%v", cart)
	}

	//合計金額がprice(1000)*2=2000であること
	if cart.Total != 2000 {
		t.Fatalf("cart.total should be 2000, got=%d", cart.Total)
	}

	//注文確定（idempotency_key を付ける）
	key := "e2e-order-key-001-" + time.Now().Format("150405.000000000")
	// 注文用の住所を作って address_id を取得する（/orders が必須になったため）
	addressID := createAddressForOrder(t, c, ctx, access)

	// 注文確定（X-Idempotency-Key はヘッダーで送る）
	order := placeOrder(t, c, ctx, access, addressID, key)

	// 注文IDが0より大きいこと
	if order.ID <= 0 {
		t.Fatalf("order id should be > 0: order=%v", order)
	}

	//注文の合計が2000であること
	if order.TotalPrice != 2000 {
		t.Fatalf("order.total_price should be 2000, got=%d", order.TotalPrice)
	}

	//明細が1件でquantity=2であること
	if len(order.Items) != 1 || order.Items[0].Quantity != 2 {
		t.Fatalf("order items mismatch: order=%v", order)
	}

	//注文後、カートが空になっていること
	cart = getCart(t, c, ctx, access)
	if len(cart.Items) != 0 || cart.Total != 0 {
		t.Fatalf("cart should be empty after order: cart=%v", cart)
	}

	//在庫が減っていること（stock 5 → 3）
	p := getProduct(t, c, ctx, productID)
	if p.Stock != 3 {
		t.Fatalf("stock should be 3 after order, got=%d", p.Stock)
	}

	//注文履歴に含まれること（/orders）
	orders := listOrders(t, c, ctx, access)

	//1件以上あること
	if len(orders) == 0 {
		t.Fatalf("orders should not be empty")
	}

	//上記で作ったorder.IDが含まれること
	found := false
	for _, o := range orders {
		if o.ID == order.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("order id not found in list: want=%d", order.ID)
	}

	//注文詳細が取れること（/orders/{id}）
	detail := getOrderDetail(t, c, ctx, access, order.ID)

	//IDが一致すること
	if detail.ID != order.ID {
		t.Fatalf("order detail id mismatch want=%d got=%d", order.ID, detail.ID)
	}

	//同じidempotency_keyで再度注文しても同じ注文が返ること
	order2 := placeOrder(t, c, ctx, access, addressID, key)
	if order2.ID != order.ID {
		t.Fatalf("idempotency violated: first=%d second=%d", order.ID, order2.ID)
	}
}

func Test_Orders_OutOfStock_WhenInventoryChangesAfterCart(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()
	access := adminLogin(t, c, ctx)
	//公開商品を作成する（stock=3）
	name := "E2E-OrderBeans-StockRace-" + time.Now().Format("20060102-150405.000000000")
	productID := createPublicProductForOrder(t, c, ctx, access, name, 3)

	//カートが空であることを確認
	cart := getCart(t, c, ctx, access)
	if len(cart.Items) != 0 {
		// カートが空前提なので、空でなければ失敗にする
		t.Fatalf("cart should be empty at start for this test: cart=%v", cart)
	}

	//カートにquantity=3を入れる（この時点では在庫と同じなので通る）
	cart = addToCart(t, c, ctx, access, productID, 3)
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 3 {
		t.Fatalf("cart add failed: cart=%v", cart)
	}

	//別購入が先に起きた想定で、管理者が在庫を0にする
	updateInventory(t, c, ctx, access, productID, 0, "simulate concurrent purchase")

	//注文確定（別キー）out of stock を期待する
	key := "e2e-order-key-oos-" + time.Now().Format("150405.000000000")
	// 注文用の住所を作る
	addressID := createAddressForOrder(t, c, ctx, access)
	// out of stock を期待
	placeOrderExpect400(t, c, ctx, access, addressID, key, "out of stock")

	//エラー時にカートが勝手に空になっていないことを確認
	cart2 := getCart(t, c, ctx, access)
	if len(cart2.Items) == 0 {
		t.Fatalf("cart should remain after failed order (expected): cart=%v", cart2)
	}
	//カートを空に戻しておく（itemsが残っている前提）
	for _, it := range cart2.Items {
		// DELETE /cart/{id}
		resp, body := c.doJSON(ctx, t, http.MethodDelete, "/cart/"+toStr(it.ID), access, nil)
		requireStatus(t, resp, http.StatusOK, body)
		_ = mustDecodeCart(t, body)
	}

	// 8) 在庫を3に戻す
	updateInventory(t, c, ctx, access, productID, 3, "cleanup")
}

// 文字列に空白が含まれていないかの軽いチェック。
func noSpace(s string) bool {
	return !strings.ContainsAny(s, " \t\r\n")
}
