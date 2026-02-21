package middleware

import (
	"net/http"

	"app/internal/repository"

	"github.com/labstack/echo/v4"
)

// JWTのtvとDBのtoken_versionの一致するか確認。
func TokenVersionGuard(userRepo repository.UserRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			//AuthJWTが入れたuser_id を取得する
			rawUserID := c.Get(CtxUserIDKey)
			userID, ok := rawUserID.(int64)
			if !ok || userID <= 0 {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//AuthJWTが入れたtoken_version(tv)を取得する
			rawTV := c.Get(CtxTokenVersionKey)
			tv, ok := rawTV.(int)
			if !ok || tv < 0 {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//DBから最新のuserを取得する
			user, err := userRepo.FindByID(c.Request().Context(), userID)
			if err != nil || user == nil {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//token_version が一致しなければ強制ログアウト扱い（401）
			if user.TokenVersion != tv {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			return next(c)
		}
	}
}
