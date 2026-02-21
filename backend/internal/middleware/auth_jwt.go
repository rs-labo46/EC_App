package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"app/internal/config"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

const (
	CtxUserIDKey       = "user_id"       // int64
	CtxUserRoleKey     = "user_role"     // string
	CtxTokenVersionKey = "token_version" // int
)

// bearerAuth用のJWT検証ミドルウェア。
func AuthJWT(cfg config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			//Authorizationヘッダを取得
			authz := c.Request().Header.Get("Authorization")
			if authz == "" {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//Bearer形式か確認してtokenを抜く
			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}
			rawToken := strings.TrimSpace(parts[1])
			if rawToken == "" {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//JWTをパースして検証する
			token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
				if t.Method != jwt.SigningMethodHS256 {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil || token == nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//claimsを取り出す
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//user_idを取り出す

			userID, err := parseUserID(claims["sub"])
			if err != nil || userID <= 0 {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//roleを取り出す（USER/ADMIN）
			role, err := parseString(claims["role"])
			if err != nil || role == "" {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//token_versionを取り出す
			tv, err := parseInt(claims["tv"])
			if err != nil || tv < 0 {
				return c.JSON(http.StatusUnauthorized, errorJSON("unauthorized"))
			}

			//contextへ保存
			c.Set(CtxUserIDKey, userID)
			c.Set(CtxUserRoleKey, role)
			c.Set(CtxTokenVersionKey, tv)

			return next(c)
		}
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func errorJSON(msg string) errorResponse {
	return errorResponse{Error: msg}
}

// user_idをint64に変換する
func parseUserID(v interface{}) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, errors.New("invalid sub")
	}
}

func parseString(v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", errors.New("invalid string")
	}
	return s, nil
}

func parseInt(v interface{}) (int, error) {
	switch t := v.(type) {
	case float64:
		return int(t), nil
	case int:
		return t, nil
	case string:
		i64, err := strconv.ParseInt(t, 10, 32)
		if err != nil {
			return 0, err
		}
		return int(i64), nil
	default:
		return 0, errors.New("invalid int")
	}
}
