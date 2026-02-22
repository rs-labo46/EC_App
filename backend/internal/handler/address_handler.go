package handler

import (
	"net/http"
	"strconv"

	"app/internal/usecase"

	"github.com/labstack/echo/v4"
)

type AddressHandler struct {
	uc *usecase.AddressUsecase
}

func NewAddressHandler(uc *usecase.AddressUsecase) *AddressHandler {
	return &AddressHandler{uc: uc}
}

func (h *AddressHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/addresses", h.List)
	g.POST("/addresses", h.Create)
	g.PATCH("/addresses/:id", h.Update)
	g.DELETE("/addresses/:id", h.Delete)
	g.POST("/addresses/:id/default", h.SetDefault)
}

func (h *AddressHandler) List(c echo.Context) error {
	userID, ok := addrGetUserID(c)
	if !ok {
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	}

	list, err := h.uc.List(c.Request().Context(), userID)
	if err != nil {
		return addrWriteUsecaseError(c, err)
	}

	return c.JSON(http.StatusOK, list)
}

func (h *AddressHandler) Create(c echo.Context) error {
	userID, ok := addrGetUserID(c)
	if !ok {
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	}

	var req usecase.AddressCreateRequest
	if err := c.Bind(&req); err != nil {
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	}

	created, err := h.uc.Create(c.Request().Context(), userID, req)
	if err != nil {
		return addrWriteUsecaseError(c, err)
	}

	return c.JSON(http.StatusOK, created)
}

func (h *AddressHandler) Update(c echo.Context) error {
	userID, ok := addrGetUserID(c)
	if !ok {
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	}

	var req usecase.AddressUpdateRequest
	if err := c.Bind(&req); err != nil {
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	}

	if err := h.uc.Update(c.Request().Context(), userID, id, req); err != nil {
		return addrWriteUsecaseError(c, err)
	}

	// Success は {message:string} に寄せる
	return c.JSON(http.StatusOK, map[string]string{"message": "updated"})
}

func (h *AddressHandler) Delete(c echo.Context) error {
	userID, ok := addrGetUserID(c)
	if !ok {
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	}

	if err := h.uc.Delete(c.Request().Context(), userID, id); err != nil {
		return addrWriteUsecaseError(c, err)
	}

	// Success は {message:string} に寄せる
	return c.JSON(http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *AddressHandler) SetDefault(c echo.Context) error {
	userID, ok := addrGetUserID(c)
	if !ok {
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	}

	if err := h.uc.SetDefault(c.Request().Context(), userID, id); err != nil {
		return addrWriteUsecaseError(c, err)
	}

	// Success は {message:string} に寄せる
	return c.JSON(http.StatusOK, map[string]string{"message": "default set"})
}

// ------- AddressHandler専用 helper（既存と衝突しないように prefix 付き） -------

func addrGetUserID(c echo.Context) (int64, bool) {
	v := c.Get("user_id")
	id, ok := v.(int64)
	return id, ok && id > 0
}

func addrWriteError(c echo.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}

func addrWriteUsecaseError(c echo.Context, err error) error {
	switch err {
	case usecase.ErrValidation:
		return addrWriteError(c, http.StatusBadRequest, "validation error")
	case usecase.ErrUnauthorized:
		return addrWriteError(c, http.StatusUnauthorized, "unauthorized")
	case usecase.ErrForbidden:
		return addrWriteError(c, http.StatusForbidden, "forbidden")
	case usecase.ErrConflict:
		return addrWriteError(c, http.StatusConflict, "conflict")
	case usecase.ErrInternal:
		return addrWriteError(c, http.StatusInternalServerError, "internal error")
	case usecase.ErrNotFound:
		return addrWriteError(c, http.StatusNotFound, "not found")
	default:
		return addrWriteError(c, http.StatusInternalServerError, "internal error")
	}
}
