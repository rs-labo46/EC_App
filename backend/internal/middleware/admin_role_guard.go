package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

//contextに入っているroleがADMINかどうかを確認します。

func AdminRoleGuard() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rawRole := c.Get(CtxUserRoleKey)
			role, ok := rawRole.(string)
			if !ok || role == "" {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//USERは拒否、ADMINだけ許可
			if role != "ADMIN" {
				return c.JSON(http.StatusForbidden, errorJSON("admin only"))
			}

			return next(c)
		}
	}
}
