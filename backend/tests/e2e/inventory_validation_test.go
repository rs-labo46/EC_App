package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func Test_AdminInventory_NegativeStock_Should400(t *testing.T) {
	c := NewTestClient(t)
	ctx := context.Background()
	access := adminLogin(t, c, ctx)

	//商品を作る（stock=5）
	uniqueName := "E2E-InvNeg-" + time.Now().Format("20060102-150405.000000000")
	create := ProductCreateRequest{
		Name:        uniqueName,
		Description: "inv validation",
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

	//一覧検索で product_id を拾う
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+uniqueName+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)

	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("product not found after create: body=%s", string(body))
	}
	productID := list.Items[0].ID

	//stock=-1 で在庫更新 → 400
	neg := InventoryUpdateRequest{Stock: -1, Reason: "negative should fail"}
	negJSON, err := json.Marshal(neg)
	if err != nil {
		t.Fatalf("json.Marshal(InventoryUpdateRequest) failed: %v", err)
	}

	resp, body = c.doJSON(ctx, t, http.MethodPut, "/admin/inventory/"+toStr(productID), access, negJSON)
	requireStatus(t, resp, http.StatusBadRequest, body)

	er := mustDecodeError(t, body)
	if strings.TrimSpace(er.Error) == "" {
		t.Fatalf("error message empty: body=%s", string(body))
	}
}
