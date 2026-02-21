package handler

import (
	"net/http"
	"strconv"

	"app/internal/usecase"

	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}
	if he, ok := usecase.AsHTTPError(err); ok {
		return c.JSON(he.Status, ErrorResponse{Error: he.Message})
	}

	//500
	return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
}

// /products の公開API
type ProductHandler struct {
	uc *usecase.ProductUsecase
}

// DI
func NewProductHandler(uc *usecase.ProductUsecase) *ProductHandler {
	return &ProductHandler{uc: uc}
}

// 公開商品のルートを登録
func (h *ProductHandler) RegisterRoutes(e *echo.Echo) {
	e.GET("/products", h.list)
	e.GET("/products/:id", h.detail)
}

func (h *ProductHandler) list(c echo.Context) error {
	// page（default 1）
	page := 1
	if v := c.QueryParam("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid page"})
		}
		page = p
	}

	// limit（default 20）
	limit := 20
	if v := c.QueryParam("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid limit"})
		}
		limit = l
	}

	q := c.QueryParam("q")
	sort := c.QueryParam("sort")

	var minPrice *int64
	if v := c.QueryParam("min_price"); v != "" {
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid min_price"})
		}
		minPrice = &x
	}

	var maxPrice *int64
	if v := c.QueryParam("max_price"); v != "" {
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid max_price"})
		}
		maxPrice = &x
	}

	out, err := h.uc.ListPublicProducts(c.Request().Context(), usecase.ListProductsInput{
		Page:     page,
		Limit:    limit,
		Q:        q,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
		Sort:     sort,
	})
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, out)
}

func (h *ProductHandler) detail(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid id"})
	}

	p, err := h.uc.GetProductDetail(c.Request().Context(), id)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(http.StatusOK, p)
}
