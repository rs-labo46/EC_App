package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"app/internal/domain/model"
	"app/internal/repository"
)

// handlerからusecaseに渡す入力
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
}

// token 形（JwtAccessToken相当）
type JwtAccessToken struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenVersion int    `json:"token_version"`
}

// handlerがJSONにして返す
type LoginOutput struct {
	User  model.User     `json:"user"`
	Token JwtAccessToken `json:"token"`
}

// handlerがCookieに詰めるために必要な値
type LoginSideEffect struct {
	PlainRefreshToken string
}

// メールまたはパスワードが違う
var ErrInvalidCredentials = errors.New("invalid credentials")

// 停止済みユーザー
var ErrUserInactive = errors.New("user is inactive")

// JWTを発行する約束
type AccessTokenIssuer interface {
	Issue(userID string, role model.Role, tokenVersion int, now time.Time) (token string, expiresAt time.Time, err error)
}

// 入力パスワードと保存したハッシュを比べる約束
type PasswordVerifier interface {
	Verify(plain string, hashed string) bool
}

type LoginUsecase struct {
	userRepo   repository.UserRepository
	rtRepo     repository.RefreshTokenRepository
	verifier   PasswordVerifier
	issuer     AccessTokenIssuer
	idGen      IDGenerator
	clock      Clock
	refreshTTL time.Duration
}

func NewLoginUsecase(
	userRepo repository.UserRepository,
	rtRepo repository.RefreshTokenRepository,
	verifier PasswordVerifier,
	issuer AccessTokenIssuer,
	idGen IDGenerator,
	clock Clock,
	refreshTTL time.Duration,
) *LoginUsecase {
	return &LoginUsecase{
		userRepo:   userRepo,
		rtRepo:     rtRepo,
		verifier:   verifier,
		issuer:     issuer,
		idGen:      idGen,
		clock:      clock,
		refreshTTL: refreshTTL,
	}
}

// ログイン処理を実行する
func (u *LoginUsecase) Execute(ctx context.Context, in LoginInput) (LoginOutput, LoginSideEffect, error) {
	var out LoginOutput
	var side LoginSideEffect

	//emailでユーザー取得
	user, err := u.userRepo.FindByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return out, side, ErrInvalidCredentials
		}
		return out, side, err
	}

	//停止ユーザーはログイン不可
	if !user.IsActive {
		return out, side, ErrUserInactive
	}

	//パスワード照合
	if ok := u.verifier.Verify(in.Password, user.Password); !ok {
		return out, side, ErrInvalidCredentials
	}

	//AccessToken発行
	now := u.clock.Now()
	accessToken, accessExp, err := u.issuer.Issue(user.ID, user.Role, user.TokenVersion, now)
	if err != nil {
		return out, side, err
	}

	//RefreshToken生成
	plainRefresh, err := generateSecureToken(32)
	if err != nil {
		return out, side, err
	}
	hash := sha256.Sum256([]byte(plainRefresh))
	refreshHash := hex.EncodeToString(hash[:])

	refresh := &model.RefreshToken{
		ID:        u.idGen.NewID(),
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: in.UserAgent,
		ExpiresAt: now.Add(u.refreshTTL),
		UsedAt:    nil,
		RevokedAt: nil,
	}

	if err := u.rtRepo.Create(ctx, refresh); err != nil {
		return out, side, err
	}

	//最終ログイン時刻更新
	user.LastLoginAt = &now
	if err := u.userRepo.Update(ctx, user); err != nil {
		return out, side, err
	}

	//出力（passwordは返さない）
	safeUser := *user
	safeUser.Password = ""

	out.User = safeUser
	out.Token = JwtAccessToken{
		AccessToken:  accessToken,
		ExpiresIn:    int(accessExp.Sub(now).Seconds()),
		TokenVersion: user.TokenVersion,
	}

	side.PlainRefreshToken = plainRefresh
	return out, side, nil
}

func generateSecureToken(bytesLen int) (string, error) {
	if bytesLen <= 0 {
		return "", fmt.Errorf("bytesLen must be positive")
	}

	// ランダムなバイト列を作る（OSが持つ安全な乱数）
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
