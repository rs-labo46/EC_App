package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

type CartItem struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
}

type CartResponse struct {
	Items []CartItem `json:"items"`
	Total int64      `json:"total"`
}

type AddCartRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int64 `json:"quantity"`
}

func mustDecodeCart(t *testing.T, body []byte) CartResponse {
	t.Helper()
	var v CartResponse
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(CartResponse) failed: %v body=%s", err, string(body))
	}
	return v
}

func Test_Cart_AddDuplicate_Patch_StockExceeded_Delete(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()
	access := adminLogin(t, c, ctx)
	clearCart(t, c, ctx, access)

	//事前準備：カート用の商品を作る（stock=5）
	uniqueName := "E2E-CartBeans-" + time.Now().Format("20060102-150405.000000000")
	create := ProductCreateRequest{
		Name:        uniqueName,
		Description: "x",
		Price:       1000,
		Stock:       5,
		IsActive:    true,
	}
	createJSON, err := json.Marshal(create)
	if err != nil {
		t.Fatalf("json.Marshal(ProductCreateRequest) failed: %v", err)
	}

	resp, body := c.doJSON(ctx, t, http.MethodPost, "/admin/products", access, createJSON)
	requireStatus(t, resp, http.StatusOK, body)

	//作った商品のIDを一覧検索で拾うか
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+uniqueName+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("product not found for cart test: body=%s", string(body))
	}
	productID := list.Items[0].ID

	//GET /cart 初回は空であるか
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/cart", access, nil)
	requireStatus(t, resp, http.StatusOK, body)

	cart := mustDecodeCart(t, body)
	if len(cart.Items) != 0 || cart.Total != 0 {
		t.Fatalf("cart should be empty: body=%s", string(body))
	}

	//POST /cartでqty=2を追加できるか
	add1 := AddCartRequest{ProductID: productID, Quantity: 2}
	add1JSON, err := json.Marshal(add1)
	if err != nil {
		t.Fatalf("json.Marshal(AddCartRequest) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPost, "/cart", access, add1JSON)
	requireStatus(t, resp, http.StatusOK, body)

	cart = mustDecodeCart(t, body)
	if len(cart.Items) != 1 {
		t.Fatalf("cart should have 1 item: body=%s", string(body))
	}
	if cart.Items[0].Quantity != 2 {
		t.Fatalf("quantity should be 2: body=%s", string(body))
	}
	itemID := cart.Items[0].ID

	//同一商品を qty=1 で追加すると合計3になるか
	add2 := AddCartRequest{ProductID: productID, Quantity: 1}
	add2JSON, err := json.Marshal(add2)
	if err != nil {
		t.Fatalf("json.Marshal(AddCartRequest) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPost, "/cart", access, add2JSON)
	requireStatus(t, resp, http.StatusOK, body)

	cart = mustDecodeCart(t, body)
	if cart.Items[0].Quantity != 3 {
		t.Fatalf("quantity should be 3 after duplicate add: body=%s", string(body))
	}

	//PATCH /cart/{id} で qty=5 に変更できるか
	patch := UpdateCartItemRequest{Quantity: 5}
	patchJSON, err := json.Marshal(patch)
	if err != nil {
		t.Fatalf("json.Marshal(UpdateCartItemRequest) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPatch, "/cart/"+toStr(itemID), access, patchJSON)
	requireStatus(t, resp, http.StatusOK, body)

	cart = mustDecodeCart(t, body)
	if cart.Items[0].Quantity != 5 {
		t.Fatalf("quantity should be 5 after patch: body=%s", string(body))
	}

	//在庫超過 qty=999は400になるか
	over := UpdateCartItemRequest{Quantity: 999}
	overJSON, err := json.Marshal(over)
	if err != nil {
		t.Fatalf("json.Marshal(UpdateCartItemRequest) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPatch, "/cart/"+toStr(itemID), access, overJSON)
	requireStatus(t, resp, http.StatusBadRequest, body)

	er := mustDecodeError(t, body)
	if er.Error != "stock exceeded" {
		t.Fatalf("error should be 'stock exceeded': body=%s", string(body))
	}

	//DELETE /cart/{id} で空に戻るか
	resp, body = c.doJSON(ctx, t, http.MethodDelete, "/cart/"+toStr(itemID), access, nil)
	requireStatus(t, resp, http.StatusOK, body)

	cart = mustDecodeCart(t, body)
	if len(cart.Items) != 0 || cart.Total != 0 {
		t.Fatalf("cart should be empty after delete: body=%s", string(body))
	}
}
