package handler

import (
	"net/http"
	"strconv"
	"time"

	"app/internal/config"
	"app/internal/middleware"
	"app/internal/repository"
	"app/internal/usecase"

	"github.com/labstack/echo/v4"
)

type AdminOrderHandler struct {
	uc *usecase.AdminOrderUsecase
}

func NewAdminOrderHandler(uc *usecase.AdminOrderUsecase) *AdminOrderHandler {
	return &AdminOrderHandler{uc: uc}
}

type OrderStatusUpdateRequest struct {
	Status string `json:"status"`
}

func (h *AdminOrderHandler) RegisterRoutes(e *echo.Echo, cfg config.Config, userRepo repository.UserRepository) {
	admin := e.Group("/admin")
	admin.Use(middleware.AuthJWT(cfg))
	admin.Use(middleware.TokenVersionGuard(userRepo))
	admin.Use(middleware.AdminRoleGuard())

	admin.GET("/orders", h.list)
	admin.PUT("/orders/:id/status", h.updateStatus)
}

func (h *AdminOrderHandler) list(c echo.Context) error {
	page := 1
	if v := c.QueryParam("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid page"})
		}
		page = p
	}

	limit := 50
	if v := c.QueryParam("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid limit"})
		}
		limit = l
	}

	status := c.QueryParam("status")

	var userID *int64
	if v := c.QueryParam("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid user_id"})
		}
		userID = &id
	}

	var fromPtr *time.Time
	if v := c.QueryParam("from"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid from"})
		}
		fromPtr = &tm
	}

	var toPtr *time.Time
	if v := c.QueryParam("to"); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid to"})
		}
		toPtr = &tm
	}

	out, err := h.uc.List(c.Request().Context(), repository.AdminOrderListFilter{
		Page:   page,
		Limit:  limit,
		Status: status,
		UserID: userID,
		From:   fromPtr,
		To:     toPtr,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *AdminOrderHandler) updateStatus(c echo.Context) error {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	var req OrderStatusUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	// ★操作した管理者IDを取得（監査ログ用）
	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	if err := h.uc.UpdateStatus(
		c.Request().Context(),
		adminID,
		orderID,
		usecase.AdminUpdateOrderStatusInput{Status: req.Status},
	); err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "updated"})
}
