package usecase

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"app/internal/domain/model"
	"app/internal/repository"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type OrderUsecase struct {
	tx        repo.TransactionManager
	addresses repository.AddressRepository
}

func NewOrderUsecase(tx repo.TransactionManager, addresses repository.AddressRepository) *OrderUsecase {
	return &OrderUsecase{tx: tx, addresses: addresses}
}

type PlaceOrderInput struct {
	AddressID      int64
	IdempotencyKey string
}

type OrderItemOutput struct {
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
}

type OrderOutput struct {
	ID         int64             `json:"id"`
	UserID     int64             `json:"user_id"`
	Status     string            `json:"status"`
	TotalPrice int64             `json:"total_price"`
	CreatedAt  time.Time         `json:"created_at"`
	Items      []OrderItemOutput `json:"items"`
}

func (u *OrderUsecase) PlaceOrder(ctx context.Context, userID int64, in PlaceOrderInput) (OrderOutput, error) {
	if userID <= 0 {
		return OrderOutput{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if in.AddressID <= 0 {
		return OrderOutput{}, NewHTTPError(http.StatusBadRequest, "invalid address_id")
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" || len(key) > 255 {
		return OrderOutput{}, NewHTTPError(http.StatusBadRequest, "invalid idempotency_key")
	}

	//address_idの存在確認＋所有チェック
	addr, err := u.addresses.FindByID(ctx, in.AddressID)
	if err != nil {
		// 住所が存在しない404
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, repo.ErrNotFound) {
			return OrderOutput{}, NewHTTPError(http.StatusNotFound, "not found")
		}
		return OrderOutput{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	//所有チェック（他人の住所なら403）
	if addr.UserID != userID {
		return OrderOutput{}, NewHTTPError(http.StatusForbidden, "forbidden")
	}

	var out OrderOutput

	//注文処理はトランザクション
	err = u.tx.WithinTx(ctx, func(r repo.TxRepos) error {
		// 同じキーなら同じ結果
		existing, found, err := r.Orders().FindByIdempotencyKey(ctx, userID, key)
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}
		if found {
			//既存注文を返す
			items, err := r.OrderItems().ListByOrderID(ctx, existing.ID)
			if err != nil {
				return NewHTTPError(http.StatusInternalServerError, "db error")
			}
			out = toOrderOutput(existing, items)
			return nil
		}

		//ACTIVEカート取得
		cart, err := r.Carts().FindActiveByUserID(ctx, userID)
		if err == repo.ErrNotFound {
			return NewHTTPError(http.StatusBadRequest, "cart empty")
		}
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		//カート明細取得
		cartItems, err := r.CartItems().ListByCartID(ctx, cart.ID)
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}
		if len(cartItems) == 0 {
			return NewHTTPError(http.StatusBadRequest, "cart empty")
		}

		//在庫を確定時に再チェックして減らす
		orderItems := make([]model.OrderItem, 0, len(cartItems))
		var total int64 = 0

		for _, ci := range cartItems {
			//商品取得
			p, err := r.Products().FindByID(ctx, ci.ProductID)
			if err == repo.ErrNotFound || !p.IsActive {
				return NewHTTPError(http.StatusBadRequest, "invalid")
			}
			if err != nil {
				return NewHTTPError(http.StatusInternalServerError, "db error")
			}

			//在庫減算（足りないなら false）
			ok, err := r.Inventory().DecreaseStockIfEnough(ctx, ci.ProductID, ci.Quantity)
			if err != nil {
				return NewHTTPError(http.StatusInternalServerError, "db error")
			}
			if !ok {
				return NewHTTPError(http.StatusBadRequest, "out of stock")
			}

			//スナップショット
			now := time.Now()
			orderItems = append(orderItems, model.OrderItem{
				ProductID:           ci.ProductID,
				ProductNameSnapshot: p.Name,
				UnitPriceSnapshot:   ci.UnitPriceSnapshot,
				Quantity:            ci.Quantity,
				CreatedAt:           now,
			})

			total += ci.UnitPriceSnapshot * ci.Quantity
		}

		// 注文作成
		now := time.Now()
		orderID, err := r.Orders().Create(ctx, model.Order{
			UserID:         userID,
			AddressID:      in.AddressID,
			Status:         model.OrderStatusPending,
			TotalPrice:     total,
			IdempotencyKey: key,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
		if err != nil {
			//競合（同時で同じキーが入った等）はもう一回検索して同じ結果を返す
			ex2, found2, err2 := r.Orders().FindByIdempotencyKey(ctx, userID, key)
			if err2 == nil && found2 {
				items2, err3 := r.OrderItems().ListByOrderID(ctx, ex2.ID)
				if err3 != nil {
					return NewHTTPError(http.StatusInternalServerError, "db error")
				}
				out = toOrderOutput(ex2, items2)
				return nil
			}
			return NewHTTPError(http.StatusBadRequest, "idempotency conflict")
		}

		//注文明細一括作成
		if err := r.OrderItems().CreateBulk(ctx, orderID, orderItems); err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		//カートをCHECKED_OUTにして、明細をクリア（再注文防止）
		if err := r.Carts().UpdateStatus(ctx, cart.ID, model.CartStatusCheckedOut); err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}
		if err := r.Carts().Clear(ctx, cart.ID); err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		created := model.Order{
			ID:         orderID,
			UserID:     userID,
			AddressID:  in.AddressID,
			Status:     model.OrderStatusPending,
			TotalPrice: total,
			CreatedAt:  now,
		}
		out = toOrderOutput(created, orderItems)
		return nil
	})

	if err != nil {
		return OrderOutput{}, err
	}
	return out, nil
}

func (u *OrderUsecase) ListMyOrders(ctx context.Context, userID int64) ([]OrderOutput, error) {
	if userID <= 0 {
		return []OrderOutput{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	//ページングでまずは固定で取る
	var outs []OrderOutput

	err := u.tx.WithinTx(ctx, func(r repo.TxRepos) error {
		orders, _, err := r.Orders().ListByUserID(ctx, userID, 1, 50)
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

func (u *OrderUsecase) GetMyOrderDetail(ctx context.Context, userID int64, orderID int64) (OrderOutput, error) {
	if userID <= 0 {
		return OrderOutput{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if orderID <= 0 {
		return OrderOutput{}, NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	var out OrderOutput

	err := u.tx.WithinTx(ctx, func(r repo.TxRepos) error {
		o, err := r.Orders().FindByID(ctx, orderID)
		if err == repo.ErrNotFound {
			return NewHTTPError(http.StatusNotFound, "not found")
		}
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}
		if o.UserID != userID {
			//他人の注文は「存在しない扱い」にする
			return NewHTTPError(http.StatusNotFound, "not found")
		}

		items, err := r.OrderItems().ListByOrderID(ctx, orderID)
		if err != nil {
			return NewHTTPError(http.StatusInternalServerError, "db error")
		}

		out = toOrderOutput(o, items)
		return nil
	})

	if err != nil {
		return OrderOutput{}, err
	}
	return out, nil
}

func toOrderOutput(o model.Order, items []model.OrderItem) OrderOutput {
	outItems := make([]OrderItemOutput, 0, len(items))
	for _, it := range items {
		outItems = append(outItems, OrderItemOutput{
			ProductID: it.ProductID,
			Name:      it.ProductNameSnapshot,
			Price:     it.UnitPriceSnapshot,
			Quantity:  it.Quantity,
		})
	}

	return OrderOutput{
		ID:         o.ID,
		UserID:     o.UserID,
		Status:     string(o.Status),
		TotalPrice: o.TotalPrice,
		CreatedAt:  o.CreatedAt,
		Items:      outItems,
	}
}
