package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// /admin/productsのリクエスト
type ProductCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int64  `json:"stock"`
	IsActive    bool   `json:"is_active"`
}

// /admin/inventory/{product_id}のリクエスト
type InventoryUpdateRequest struct {
	Stock  int64  `json:"stock"`
	Reason string `json:"reason"`
}

type Product struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int64  `json:"stock"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProductList struct {
	Items []Product `json:"items"`
	Total int64     `json:"total"`
	Page  int       `json:"page,omitempty"`
	Limit int       `json:"limit,omitempty"`
}

func mustDecodeProductList(t *testing.T, body []byte) ProductList {
	t.Helper()
	var v ProductList
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(ProductList) failed: %v body=%s", err, string(body))
	}
	return v
}

func mustDecodeProduct(t *testing.T, body []byte) Product {
	t.Helper()
	var v Product
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("json.Unmarshal(Product) failed: %v body=%s", err, string(body))
	}
	return v
}

func Test_Product_AdminCRUD_PublicRead_InventoryUpdate(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()
	access := adminLogin(t, c, ctx)

	//商品作成
	uniqueName := "E2E-Beans-" + time.Now().Format("20060102-150405.000000000")

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
	_ = mustDecodeSuccess(t, body)

	//公開一覧で検索して作った商品が見つかるか
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+uniqueName+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("product not found in list: body=%s", string(body))
	}

	created := list.Items[0]
	if created.Name != uniqueName {
		t.Fatalf("name mismatch want=%s got=%s", uniqueName, created.Name)
	}
	productID := created.ID

	//公開詳細が200で返るか
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products/"+toStr(productID), "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	detail := mustDecodeProduct(t, body)
	if detail.ID != productID {
		t.Fatalf("id mismatch want=%d got=%d", productID, detail.ID)
	}

	//商品更新
	update := ProductCreateRequest{
		Name:        uniqueName + "+",
		Description: "new",
		Price:       1200,
		Stock:       10,
		IsActive:    true,
	}
	updateJSON, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("json.Marshal(update) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPut, "/admin/products/"+toStr(productID), access, updateJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//在庫更新
	inv := InventoryUpdateRequest{Stock: 9, Reason: "e2e"}
	invJSON, err := json.Marshal(inv)
	if err != nil {
		t.Fatalf("json.Marshal(inv) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPut, "/admin/inventory/"+toStr(productID), access, invJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//公開詳細でstockが反映されていること
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products/"+toStr(productID), "", nil)
	requireStatus(t, resp, http.StatusOK, body)
	afterInv := mustDecodeProduct(t, body)
	if afterInv.Stock != 9 {
		t.Fatalf("stock mismatch want=9 got=%d", afterInv.Stock)
	}

	//削除
	resp, body = c.doJSON(ctx, t, http.MethodDelete, "/admin/products/"+toStr(productID), access, nil)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//削除後は公開詳細が404
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products/"+toStr(productID), "", nil)
	requireStatus(t, resp, http.StatusNotFound, body)

	er := mustDecodeError(t, body)
	if strings.TrimSpace(er.Error) == "" {
		t.Fatalf("error message empty: body=%s", string(body))
	}
}
