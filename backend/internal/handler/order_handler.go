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

type OrderHandler struct {
	uc *usecase.OrderUsecase
}

func NewOrderHandler(uc *usecase.OrderUsecase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

type OrderCreateRequest struct {
	AddressID      int64  `json:"address_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (h *OrderHandler) RegisterRoutes(e *echo.Echo, cfg config.Config, userRepo repository.UserRepository) {
	g := e.Group("/orders")
	g.Use(middleware.AuthJWT(cfg))
	g.Use(middleware.TokenVersionGuard(userRepo))

	g.POST("", h.create)
	g.GET("", h.list)
	g.GET("/:id", h.detail)
}

func (h *OrderHandler) create(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	var req OrderCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	//二重送信防止キーはヘッダーから受け取る（bodyには入れない）

	idemKey := c.Request().Header.Get("X-Idempotency-Key")

	out, err := h.uc.PlaceOrder(c.Request().Context(), userID, usecase.PlaceOrderInput{
		AddressID:      req.AddressID,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *OrderHandler) list(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	out, err := h.uc.ListMyOrders(c.Request().Context(), userID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, out)
}

func (h *OrderHandler) detail(c echo.Context) error {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	out, err := h.uc.GetMyOrderDetail(c.Request().Context(), userID, id)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, out)
}
