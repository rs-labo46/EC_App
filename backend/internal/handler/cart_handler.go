package handler

import (
	"net/http"
	"strconv"

	"app/internal/config"
	"app/internal/middleware"
	"app/internal/repository"
	"app/internal/usecase"

	"github.com/labstack/echo/v4"
)

// /cartのHTTP
type CartHandler struct {
	uc *usecase.CartUsecase
}

// DI
func NewCartHandler(uc *usecase.CartUsecase) *CartHandler {
	return &CartHandler{uc: uc}
}

type AddCartRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int64 `json:"quantity"`
}

// /cart, /cart/{id} を登録
func (h *CartHandler) RegisterRoutes(e *echo.Echo, cfg config.Config, userRepo repository.UserRepository) {
	g := e.Group("/cart")
	g.Use(middleware.AuthJWT(cfg))
	g.Use(middleware.TokenVersionGuard(userRepo))

	g.GET("", h.getCart)
	g.POST("", h.addToCart)
	g.PATCH("/:id", h.patchItem)
	g.DELETE("/:id", h.deleteItem)
}

func (h *CartHandler) getCart(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	out, err := h.uc.GetCart(c.Request().Context(), userID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *CartHandler) addToCart(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	var req AddCartRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	out, err := h.uc.AddToCart(c.Request().Context(), userID, usecase.AddCartInput{
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *CartHandler) patchItem(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	var req UpdateCartItemRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	out, err := h.uc.UpdateCartItem(c.Request().Context(), userID, itemID, usecase.UpdateCartItemInput{
		Quantity: req.Quantity,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *CartHandler) deleteItem(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	itemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	out, err := h.uc.DeleteCartItem(c.Request().Context(), userID, itemID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}
