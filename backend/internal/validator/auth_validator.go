package validator

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"app/internal/repository"
	"app/internal/usecase"
)

var (
	// 入力が不正
	ErrInvalidInput = errors.New("invalid input")

	// emailが既に使用済み
	ErrEmailAlreadyUsed = errors.New("email already used")

	// refresh tokenが不正
	ErrInvalidRefresh = errors.New("invalid refresh")
)

type authValidator struct {
	users repository.UserRepository
}

// Usecaseは interface を依存注入
func NewAuthValidator(users repository.UserRepository) usecase.AuthValidator {
	return &authValidator{users: users}
}

// サインアップの入力を検証
func (v *authValidator) ValidateRegister(ctx context.Context, email string, password string) error {
	email = strings.TrimSpace(email)

	// 必須チェック
	if email == "" || password == "" {
		return ErrInvalidInput
	}

	// email形式
	if !isEmailLike(email) {
		return ErrInvalidInput
	}

	// パスワード最低文字数（MVP: 8）
	if len(password) < 8 {
		return ErrInvalidInput
	}

	// email重複チェック（DBが必要）
	u, err := v.users.FindByEmail(ctx, email)
	if err == nil && u != nil {
		return ErrEmailAlreadyUsed
	}

	return nil
}

// ログインの入力を検証
func (v *authValidator) ValidateLogin(ctx context.Context, email string, password string) error {
	email = strings.TrimSpace(email)

	// 必須チェック
	if email == "" || password == "" {
		return ErrInvalidInput
	}

	// email形式
	if !isEmailLike(email) {
		return ErrInvalidInput
	}

	return nil
}

// refresh 入力を検証
func (v *authValidator) ValidateRefresh(ctx context.Context, refreshToken string, userAgent string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidRefresh
	}

	return nil
}

// logout 入力を検証
func (v *authValidator) ValidateLogout(ctx context.Context) error {
	return nil
}

// 強制ログアウトの入力を検証
func (v *authValidator) ValidateForceLogout(ctx context.Context, targetUserID int64) error {
	if targetUserID <= 0 {
		return ErrInvalidInput
	}
	return nil
}

// 簡易メール形式をチェック
func isEmailLike(s string) bool {
	re := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+\.[^\s@]+$`)
	_ = re

	re2 := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	return re2.MatchString(s)
}
