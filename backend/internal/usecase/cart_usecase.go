package usecase

import (
	repo "app/internal/repository"
	"context"
	"net/http"
)

// CartUsecase は /cart の業務ロジックです。
// Repositoryは仕様書どおり、Cart と CartItem を分離して受け取ります。
type CartUsecase struct {
	cartRepo     repo.CartRepository
	cartItemRepo repo.CartItemRepository
	productRepo  repo.ProductRepository
}

func NewCartUsecase(
	cartRepo repo.CartRepository,
	cartItemRepo repo.CartItemRepository,
	productRepo repo.ProductRepository,
) *CartUsecase {
	return &CartUsecase{
		cartRepo:     cartRepo,
		cartItemRepo: cartItemRepo,
		productRepo:  productRepo,
	}
}

// CartItemResponse は OASの CartItem に合わせます。
// price は unit_price_snapshot（追加時点の価格）を返します。
type CartItemResponse struct {
	ID        int64  `json:"id"`
	ProductID int64  `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
	Quantity  int64  `json:"quantity"`
}

// CartResponse は OASの CartResponse に合わせます。
type CartResponse struct {
	Items []CartItemResponse `json:"items"`
	Total int64              `json:"total"`
}

// OAS: AddCartRequest
type AddCartInput struct {
	ProductID int64
	Quantity  int64
}

// OAS: UpdateCartItemRequest
type UpdateCartItemInput struct {
	Quantity int64
}

// GetCart はカート取得（無ければACTIVEを作って空を返す）。
func (u *CartUsecase) GetCart(ctx context.Context, userID int64) (CartResponse, error) {
	if userID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	cart, err := u.cartRepo.GetOrCreateActiveByUserID(ctx, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	return u.buildCartResponse(ctx, cart.ID)
}

// AddToCart はカートに追加（同一商品は数量加算）。
func (u *CartUsecase) AddToCart(ctx context.Context, userID int64, in AddCartInput) (CartResponse, error) {
	if userID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if in.ProductID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid product_id")
	}
	if in.Quantity < 1 {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid quantity")
	}

	// ACTIVEカート取得（無ければ作成）
	cart, err := u.cartRepo.GetOrCreateActiveByUserID(ctx, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	// 商品チェック（公開のみ）
	p, err := u.productRepo.FindByID(ctx, in.ProductID)
	if err == repo.ErrNotFound {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid")
	}
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	if !p.IsActive {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid")
	}

	// 既存数量を仕様どおり ListByCartID で調べる（FindByCartAndProductは追加しない）
	items, err := u.cartItemRepo.ListByCartID(ctx, cart.ID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	var existingQty int64 = 0
	for _, it := range items {
		if it.ProductID == in.ProductID {
			existingQty = it.Quantity
			break
		}
	}

	newQty := existingQty + in.Quantity
	if newQty > p.Stock {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "stock exceeded")
	}

	// Upsert（同一商品は加算）
	// unit_price_snapshot は「追加時点の価格」を渡す
	if err := u.cartItemRepo.UpsertByCartAndProduct(ctx, cart.ID, in.ProductID, in.Quantity, p.Price); err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	return u.buildCartResponse(ctx, cart.ID)
}

// 数量変更（所有チェック＋在庫チェック）。
func (u *CartUsecase) UpdateCartItem(ctx context.Context, userID int64, cartItemID int64, in UpdateCartItemInput) (CartResponse, error) {
	if userID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if cartItemID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	if in.Quantity < 1 {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid quantity")
	}

	owned, err := u.cartItemRepo.IsOwnedByUser(ctx, cartItemID, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	if !owned {
		return CartResponse{}, NewHTTPError(http.StatusNotFound, "not found")
	}

	item, err := u.cartItemRepo.FindByID(ctx, cartItemID)
	if err == repo.ErrNotFound {
		return CartResponse{}, NewHTTPError(http.StatusNotFound, "not found")
	}
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	//商品の在庫チェック
	p, err := u.productRepo.FindByID(ctx, item.ProductID)
	if err == repo.ErrNotFound {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid")
	}
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	if !p.IsActive {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid")
	}
	if in.Quantity > p.Stock {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "stock exceeded")
	}

	if err := u.cartItemRepo.UpdateQuantity(ctx, cartItemID, in.Quantity); err != nil {
		if err == repo.ErrNotFound {
			return CartResponse{}, NewHTTPError(http.StatusNotFound, "not found")
		}
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	//ACTIVEカートを取得して返却
	cart, err := u.cartRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	return u.buildCartResponse(ctx, cart.ID)
}

// 明細削除
func (u *CartUsecase) DeleteCartItem(ctx context.Context, userID int64, cartItemID int64) (CartResponse, error) {
	if userID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if cartItemID <= 0 {
		return CartResponse{}, NewHTTPError(http.StatusBadRequest, "invalid id")
	}

	owned, err := u.cartItemRepo.IsOwnedByUser(ctx, cartItemID, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	if !owned {
		return CartResponse{}, NewHTTPError(http.StatusNotFound, "not found")
	}

	if err := u.cartItemRepo.DeleteByID(ctx, cartItemID); err != nil {
		if err == repo.ErrNotFound {
			return CartResponse{}, NewHTTPError(http.StatusNotFound, "not found")
		}
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	cart, err := u.cartRepo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}
	return u.buildCartResponse(ctx, cart.ID)
}

// cartIDの明細をまとめてCartResponseを作る。
func (u *CartUsecase) buildCartResponse(ctx context.Context, cartID int64) (CartResponse, error) {
	items, err := u.cartItemRepo.ListByCartID(ctx, cartID)
	if err != nil {
		return CartResponse{}, NewHTTPError(http.StatusInternalServerError, "db error")
	}

	respItems := make([]CartItemResponse, 0, len(items))
	var total int64 = 0

	for _, it := range items {
		p, err := u.productRepo.FindByID(ctx, it.ProductID)
		if err != nil {
			continue
		}
		if !p.IsActive {
			continue
		}

		respItems = append(respItems, CartItemResponse{
			ID:        it.ID,
			ProductID: it.ProductID,
			Name:      p.Name,
			Price:     it.UnitPriceSnapshot,
			Quantity:  it.Quantity,
		})

		total += it.UnitPriceSnapshot * it.Quantity
	}

	return CartResponse{Items: respItems, Total: total}, nil
}
