package server

import (
	"app/internal/handler"
	"net/http"
)

func Start(addr string, authH *handler.AuthHandler) error {

	RegisterRoutes(authH)
	return http.ListenAndServe(addr, nil)
}
