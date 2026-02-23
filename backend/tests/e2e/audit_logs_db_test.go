package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// audit_logsのactionを拾うため
type auditRow struct {
	Action string
}

// DB接続文字列を環境変数から読む。
func auditTestDSN() string {
	if v := os.Getenv("TEST_DATABASE_DSN"); v != "" {
		return v
	}
	return "postgres://myuser:mypassword@localhost:5433/mydb?sslmode=disable"
}

func Test_AuditLogs_UpdateStock_And_UpdateOrderStatus_AreRecorded(t *testing.T) {
	// 1) DB接続
	dsn := auditTestDSN()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	//APIで監査ログが発生する行動を起こす
	c := NewTestClient(t)
	access := adminLogin(t, c, ctx)

	//商品作成
	uniqueName := "E2E-Audit-" + time.Now().Format("20060102-150405.000000000")
	create := ProductCreateRequest{
		Name:        uniqueName,
		Description: "audit test",
		Price:       1000,
		Stock:       5,
		IsActive:    true,
	}
	createJSON, _ := json.Marshal(create)
	resp, body := c.doJSON(ctx, t, http.MethodPost, "/admin/products", access, createJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	// product_id を拾う
	resp, body = c.doJSON(ctx, t, http.MethodGet, "/products?page=1&limit=20&q="+uniqueName+"&sort=new", "", nil)
	requireStatus(t, resp, http.StatusOK, body)
	list := mustDecodeProductList(t, body)
	if len(list.Items) == 0 {
		t.Fatalf("product not found after create: body=%s", string(body))
	}
	productID := list.Items[0].ID

	//在庫更新（UPDATE_STOCK が出る想定）
	inv := InventoryUpdateRequest{Stock: 4, Reason: "audit-update-stock"}
	invJSON, _ := json.Marshal(inv)
	resp, body = c.doJSON(ctx, t, http.MethodPut, "/admin/inventory/"+toStr(productID), access, invJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//注文作成→管理者でステータス更新（UPDATE_ORDER_STATUS が出る想定）
	clearCart(t, c, ctx, access)
	addToCart(t, c, ctx, access, productID, 1)
	addressID := createAddressForOrder(t, c, ctx, access)

	key := "e2e-audit-order-key-" + time.Now().Format("150405.000000000")
	order := placeOrder(t, c, ctx, access, addressID, key)
	if order.ID <= 0 {
		t.Fatalf("order id should be > 0: order=%v", order)
	}

	//管理者で PENDING → SHIPPED（UPDATE_ORDER_STATUS を出す）
	req := OrderStatusUpdateRequest{Status: "SHIPPED"}
	reqJSON, _ := json.Marshal(req)
	resp, body = c.doJSON(ctx, t, http.MethodPut, "/admin/orders/"+toStr(order.ID)+"/status", access, reqJSON)
	requireStatus(t, resp, http.StatusOK, body)
	_ = mustDecodeSuccess(t, body)

	//DBで audit_logs を確認
	rows, err := db.QueryContext(ctx, `
		select action
		from audit_logs
		order by id desc
		limit 50
	`)
	if err != nil {
		t.Fatalf("query audit_logs failed: %v (dsn=%s)", err, dsn)
	}
	defer func() { _ = rows.Close() }()

	actions := make([]string, 0, 50)
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatalf("rows.Scan failed: %v", err)
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err: %v", err)
	}

	//UPDATE_STOCK / UPDATE_ORDER_STATUS が含まれること
	hasStock := false
	hasOrder := false
	for _, a := range actions {
		if a == "UPDATE_STOCK" {
			hasStock = true
		}
		if a == "UPDATE_ORDER_STATUS" {
			hasOrder = true
		}
	}

	if !hasStock || !hasOrder {
		t.Fatalf("audit logs missing. hasStock=%v hasOrder=%v actions=%s",
			hasStock, hasOrder, strings.Join(actions, ","))
	}
}
