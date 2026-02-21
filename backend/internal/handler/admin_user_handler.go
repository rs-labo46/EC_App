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

type AdminUserHandler struct {
	cfg      config.Config
	userRepo repository.UserRepository
	uc       *usecase.AuthUsecase
}

func NewAdminUserHandler(cfg config.Config, userRepo repository.UserRepository, uc *usecase.AuthUsecase) *AdminUserHandler {
	return &AdminUserHandler{cfg: cfg, userRepo: userRepo, uc: uc}
}

func (h *AdminUserHandler) RegisterRoutes(e *echo.Echo) {
	// ★ /admin 配下は全部「JWT必須 + token_version一致 + ADMIN限定」
	admin := e.Group(
		"/admin",
		middleware.AuthJWT(h.cfg),
		middleware.TokenVersionGuard(h.userRepo),
		middleware.AdminRoleGuard(),
	)

	admin.POST("/users/:id/force-logout", h.ForceLogout)
}

func (h *AdminUserHandler) ForceLogout(c echo.Context) error {
	idStr := c.Param("id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || userID <= 0 {
		return c.JSON(http.StatusBadRequest, errorJSON("invalid user_id"))
	}

	res, uerr := h.uc.ForceLogout(c.Request().Context(), userID)
	if uerr != nil {
		return c.JSON(http.StatusInternalServerError, errorJSON("internal error"))
	}

	return c.JSON(http.StatusOK, res)
}
