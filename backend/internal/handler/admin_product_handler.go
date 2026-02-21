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

// SuccessResponse は OAS の Success { message: string } の形に寄せます。
type SuccessResponse struct {
	Message string `json:"message"`
}

// ProductCreateRequest は OAS の ProductCreate に合わせます。
type ProductCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int64  `json:"stock"`
	IsActive    bool   `json:"is_active"`
}

// InventoryUpdateRequest は在庫更新の入力です。
type InventoryUpdateRequest struct {
	Stock  int64  `json:"stock"`
	Reason string `json:"reason"`
}

// /admin/products と /admin/inventory をまとめる
type AdminProductHandler struct {
	uc *usecase.ProductUsecase
}

// DI
func NewAdminProductHandler(uc *usecase.ProductUsecase) *AdminProductHandler {
	return &AdminProductHandler{uc: uc}
}

// adminを登録
func (h *AdminProductHandler) RegisterRoutes(e *echo.Echo, cfg config.Config, userRepo repository.UserRepository) {
	admin := e.Group("/admin")

	admin.Use(middleware.AuthJWT(cfg))
	admin.Use(middleware.TokenVersionGuard(userRepo))
	admin.Use(middleware.AdminRoleGuard())

	admin.POST("/products", h.createProduct)
	admin.PUT("/products/:id", h.updateProduct)
	admin.DELETE("/products/:id", h.deleteProduct)
	admin.PUT("/inventory/:product_id", h.updateInventory)
}

func (h *AdminProductHandler) createProduct(c echo.Context) error {
	var req ProductCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	_, err := h.uc.AdminCreateProduct(
		c.Request().Context(),
		adminID,
		usecase.AdminCreateProductInput{
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Stock:       req.Stock,
			IsActive:    req.IsActive,
		},
	)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "created"})
}

func (h *AdminProductHandler) updateProduct(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	var req ProductCreateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	err = h.uc.AdminUpdateProduct(
		c.Request().Context(),
		adminID,
		id,
		usecase.AdminCreateProductInput{
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Stock:       req.Stock,
			IsActive:    req.IsActive,
		},
	)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "updated"})
}

func (h *AdminProductHandler) deleteProduct(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	if err := h.uc.AdminDeleteProduct(c.Request().Context(), adminID, id); err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "deleted"})
}

func (h *AdminProductHandler) updateInventory(c echo.Context) error {
	productID, err := strconv.ParseInt(c.Param("product_id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid product_id"})
	}

	var req InventoryUpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid body"})
	}

	adminID, ok := getUserIDFromContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	if err := h.uc.AdminUpdateInventory(
		c.Request().Context(),
		adminID,
		productID,
		req.Stock,
		req.Reason,
	); err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, SuccessResponse{Message: "stock updated"})
}

//middleware.AuthJWT が c.Set("user_id", int64) した値を取り出す

func getUserIDFromContext(c echo.Context) (int64, bool) {
	v := c.Get("user_id")
	if v == nil {
		return 0, false
	}

	id, ok := v.(int64)
	if !ok {
		return 0, false
	}

	return id, true
}
