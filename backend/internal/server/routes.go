package server

import (
	"app/internal/handler"
	"net/http"
)

func RegisterRoutes(authH *handler.AuthHandler) {
	http.HandleFunc("/auth/register", authH.Register)
	http.HandleFunc("/auth/login", authH.Login)
}
