package usecase

import (
	"context"
	"net/http"
	"strings"
	"time"

	"app/internal/domain/model"
	repo "app/internal/repository"
)

type AdminOrderUsecase struct {
	tx        repo.TransactionManager
	auditRepo repo.AuditLogRepository
}

func NewAdminOrderUsecase(tx repo.TransactionManager, auditRepo repo.AuditLogRepository) *AdminOrderUsecase {
	return &AdminOrderUsecase{tx: tx, auditRepo: auditRepo}
}

type AdminUpdateOrderStatusInput struct {
	Status string
}

// 注文一覧（
func (u *AdminOrderUsecase) List(ctx context.Context, f repo.AdminOrderListFilter) ([]OrderOutput, error) {
	// page/limitの最低限チェック
	if f.Page < 1 {
		return []OrderOutput{}, NewHTTPError(http.StatusBadRequest, "invalid page")
	}
	if f.Limit < 1 || f.Limit > 100 {
		return []OrderOutput{}, NewHTTPError(http.StatusBadRequest, "invalid limit")
	}

	var outs []OrderOutput

	err := u.tx.WithinTx(ctx, func(r repo.TxRepos) error {
		orders, _, err := r.Orders().ListAdmin(ctx, f)
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		outs = make([]OrderOutput, 0, len(orders))
		for _, o := range orders {
			items, err := r.OrderItems().ListByOrderID(ctx, o.ID)
			if err != nil {
				return NewHTTPError(http.StatusInternalServerError, "db error")
			}
			outs = append(outs, toOrderOutput(o, items))
		}
		return nil
	})

	if err != nil {
		return []OrderOutput{}, err
	}
	return outs, nil
}

// ステータス更新（CANCELED なら在庫戻し)
func (u *AdminOrderUsecase) UpdateStatus(ctx context.Context, actorAdminUserID int64, orderID int64, in AdminUpdateOrderStatusInput) error {
	if actorAdminUserID <= 0 {
		return NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if orderID <= 0 {
		return NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	newStatus := strings.TrimSpace(in.Status)
	switch newStatus {
	case "PENDING", "PAID", "SHIPPED", "CANCELED":
		// OK
	default:
		return NewHTTPError(http.StatusBadRequest, "invalid status")
	}

	return u.tx.WithinTx(ctx, func(r repo.TxRepos) error {
		// 注文取得
		o, err := r.Orders().FindByID(ctx, orderID)
		if err == repo.ErrNotFound {
			return NewHTTPError(http.StatusNotFound, "not found")
		}
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		// すでに同じなら何もしない（200）
		if string(o.Status) == newStatus {
			return nil
		}
		// 終端ガード
		if o.Status == model.OrderStatusCanceled {
			return NewHTTPError(http.StatusBadRequest, "cannot change canceled order")
		}
		if o.Status == model.OrderStatusShipped {
			return NewHTTPError(http.StatusBadRequest, "cannot change shipped order")
		}

		// newStatusがCANCELEDのときだけ在庫戻し
		if newStatus == "CANCELED" {
			if o.Status == model.OrderStatusPending || o.Status == model.OrderStatusPaid {
				items, err := r.OrderItems().ListByOrderID(ctx, orderID)
				if err != nil {
					return NewHTTPError(http.StatusInternalServerError, "db error")
				}

				for _, it := range items {
					if err := r.Inventory().IncreaseStock(ctx, it.ProductID, it.Quantity); err != nil {
						return NewHTTPError(http.StatusInternalServerError, "db error")
					}
				}
			}
		}

		// ステータス更新
		beforeStatus := string(o.Status)
		if err := r.Orders().UpdateStatus(ctx, orderID, model.OrderStatus(newStatus)); err != nil {
			if err == repo.ErrNotFound {
				return NewHTTPError(http.StatusNotFound, "not found")
			}
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		// ★監査ログ（UPDATE_ORDER_STATUS）
		beforeJSON := `{"status":"` + beforeStatus + `"}`
		afterJSON := `{"status":"` + newStatus + `"}`
		if err := u.auditRepo.Create(ctx, model.AuditLog{
			ActorUserID:  actorAdminUserID,
			Action:       model.AuditActionUpdateOrderStatus,
			ResourceType: model.AuditResourceOrder,
			ResourceID:   orderID,
			BeforeJSON:   beforeJSON,
			AfterJSON:    afterJSON,
			CreatedAt:    time.Now(),
		}); err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		return nil
	})
}

// 期間パラメータでtime.Timeが必要なら、handlerでtime.Parseしてここに入れる
func parseDateTimeRFC3339(s string) (*time.Time, bool) {
	if strings.TrimSpace(s) == "" {
		return nil, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, false
	}
	return &t, true
}
