package usecase

import (
	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/repository"

	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	//400 入力不足
	ErrValidation = errors.New("validation error")
	//401 認証失敗
	ErrUnauthorized = errors.New("unauthorized")
	//403　権限
	ErrForbidden = errors.New("forbidden ")
	//401 再利用されてしまっている
	ErrSecurityIncident = errors.New("security incident")
	//競合
	ErrConflict = errors.New("conflict")
	//500
	ErrInternal = errors.New("internal error")
)

// accesstokenの有効期限
const accessTokenTTL = 15 * time.Minute

// refreshtokenの有効期限
const refreshTokenTTL = 30 * 24 * time.Hour

// usecaseがValidatorInterfaceに依存する約束
type AuthValidator interface {
	ValidateRegister(ctx context.Context, email string, password string) error
	ValidateLogin(ctx context.Context, email string, password string) error
	ValidateRefresh(ctx context.Context, refreshToken string, userAgent string) error
	ValidateLogout(ctx context.Context) error
	ValidateForceLogout(ctx context.Context, targetUserID int64) error
}

type UserDTO struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int    `json:"token_version"`
	IsActive     bool   `json:"is_active"`
}

type JwtAccessTokenDTO struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenVersion int    `json:"token_version"`
}

type AuthRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthRegisterResponse struct {
	User UserDTO `json:"user"`
}

type AuthLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthLoginResponse struct {
	User  UserDTO           `json:"user"`
	Token JwtAccessTokenDTO `json:"token"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ForceLogoutResponse struct {
	UserID          int64 `json:"user_id"`
	NewTokenVersion int   `json:"new_token_version"`
}

type LoginResult struct {
	Body              AuthLoginResponse
	RefreshTokenPlain string
	CsrfTokenPlain    string
}

type RefreshResult struct {
	Body              JwtAccessTokenDTO
	RefreshTokenPlain string
	CsrfTokenPlain    string
}

type AuthUsecase struct {
	cfg       config.Config
	users     repository.UserRepository
	rtRepo    repository.RefreshTokenRepository
	validator AuthValidator
}

func NewAuthUsecase(
	cfg config.Config,
	users repository.UserRepository,
	rtRepo repository.RefreshTokenRepository,
	validator AuthValidator,
) *AuthUsecase {
	return &AuthUsecase{
		cfg:       cfg,
		users:     users,
		rtRepo:    rtRepo,
		validator: validator,
	}
}

func (u *AuthUsecase) Register(ctx context.Context, req AuthRegisterRequest) (*AuthRegisterResponse, error) {
	//入力検証（validatorに寄せる）
	if err := u.validator.ValidateRegister(ctx, req.Email, req.Password); err != nil {
		return nil, err
	}

	//パスワードは必ずハッシュ化して保存（平文保存しない）
	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, ErrInternal
	}

	//ユーザー作成
	user := &model.User{
		Email:        req.Email,
		PasswordHash: string(pwHash),
		Role:         model.RoleUser,
		TokenVersion: 0,
		IsActive:     true,
	}

	//保存（email重複などはrepo/validatorで弾く）
	if err := u.users.Create(ctx, user); err != nil {
		// ここで unique 違反を検知できるなら ErrConflict にしたい
		return nil, ErrConflict
	}

	//DTOに変換して返す
	userDTO, err := toUserDTOOpenAPI(user)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthRegisterResponse{
		User: userDTO,
	}, nil
}

func (u *AuthUsecase) Login(ctx context.Context, req AuthLoginRequest, userAgent string, ip string) (*LoginResult, error) {
	// 1) 入力検証
	if err := u.validator.ValidateLogin(ctx, req.Email, req.Password); err != nil {
		return nil, err
	}

	//ユーザー取得
	user, err := u.users.FindByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, ErrUnauthorized
	}

	//停止ユーザーはログイン不可
	if !user.IsActive {
		return nil, ErrForbidden
	}

	//パスワード照合（bcrypt）
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrUnauthorized
	}

	//last_login更新
	now := time.Now()
	user.LastLoginAt = &now
	_ = u.users.Update(ctx, user)

	//access token発行（JwtAccessToken）
	accessToken, expiresIn, err := u.issueAccessToken(user)
	if err != nil {
		return nil, ErrInternal
	}

	//refresh token発行（DBにはhash保存）
	refreshPlain, refreshHash, err := newRandomTokenAndHash()
	if err != nil {
		return nil, ErrInternal
	}

	//RefreshTokenモデルにIPを足してここで保存。
	_ = ip

	rt := &model.RefreshToken{
		ID:        uuid.NewString(), // refresh token 自体のID
		UserID:    user.ID,
		TokenHash: refreshHash,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
		UsedAt:    nil,
		RevokedAt: nil,
	}

	if err := u.rtRepo.Create(ctx, rt); err != nil {
		return nil, ErrInternal
	}

	//CSRFtoken
	csrfPlain, _, err := newRandomTokenAndHash()
	if err != nil {
		return nil, ErrInternal
	}

	//UserDTO
	userDTO, err := toUserDTOOpenAPI(user)
	if err != nil {
		return nil, ErrInternal
	}

	res := &LoginResult{
		Body: AuthLoginResponse{
			User: userDTO,
			Token: JwtAccessTokenDTO{
				AccessToken:  accessToken,
				ExpiresIn:    expiresIn,
				TokenVersion: user.TokenVersion,
			},
		},
		RefreshTokenPlain: refreshPlain,
		CsrfTokenPlain:    csrfPlain,
	}

	return res, nil
}

// model.UserをAPI返却用DTOに変換。
func toUserDTO(u *model.User) UserDTO {
	return UserDTO{
		ID:           u.ID,
		Email:        u.Email,
		Role:         string(u.Role),
		TokenVersion: u.TokenVersion,
		IsActive:     u.IsActive,
	}
}
func (u *AuthUsecase) Me(ctx context.Context, userID int64) (*UserDTO, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}

	user, err := u.users.FindByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUnauthorized
	}

	if !user.IsActive {
		return nil, ErrForbidden
	}

	dto := toUserDTO(user)
	return &dto, nil
}

func (u *AuthUsecase) Refresh(ctx context.Context, refreshTokenPlain string, userAgent string, ip string) (*RefreshResult, error) {
	//入力検証
	if err := u.validator.ValidateRefresh(ctx, refreshTokenPlain, userAgent); err != nil {
		return nil, err
	}

	//DB照合
	tokenHash := hashToken(refreshTokenPlain)

	rt, err := u.rtRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil || rt == nil {
		return nil, ErrUnauthorized
	}

	//期限切れ
	if rt.ExpiresAt.Before(time.Now()) {
		_ = u.rtRepo.DeleteByID(ctx, rt.ID)
		return nil, ErrUnauthorized
	}

	//revoked
	if rt.RevokedAt != nil {
		return nil, ErrUnauthorized
	}

	//used済みが来たら replay → 全削除
	if rt.UsedAt != nil {
		_ = u.rtRepo.DeleteAllByUserID(ctx, rt.UserID)
		return nil, ErrSecurityIncident
	}

	// 6) user_agent違い（再認証扱い。全削除）
	if userAgent != "" && rt.UserAgent != "" && userAgent != rt.UserAgent {
		_ = u.rtRepo.DeleteAllByUserID(ctx, rt.UserID)
		return nil, ErrSecurityIncident
	}

	//user取得
	user, err := u.users.FindByID(ctx, rt.UserID)
	if err != nil || user == nil {
		return nil, ErrUnauthorized
	}
	if !user.IsActive {
		return nil, ErrForbidden
	}

	//旧tokenをusedにする
	if err := u.rtRepo.MarkUsed(ctx, rt.ID); err != nil {
		_ = u.rtRepo.DeleteAllByUserID(ctx, rt.UserID)
		return nil, ErrSecurityIncident
	}

	//新tokenを作って保存
	newPlain, newHash, err := newRandomTokenAndHash()
	if err != nil {
		return nil, ErrInternal
	}

	_ = ip

	newRT := &model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: newHash,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}

	if err := u.rtRepo.Create(ctx, newRT); err != nil {
		return nil, ErrInternal
	}

	//access再発行（JwtAccessToken）
	accessToken, expiresIn, err := u.issueAccessToken(user)
	if err != nil {
		return nil, ErrInternal
	}

	//CSRFも更新
	csrfPlain, _, err := newRandomTokenAndHash()
	if err != nil {
		return nil, ErrInternal
	}

	return &RefreshResult{
		Body: JwtAccessTokenDTO{
			AccessToken:  accessToken,
			ExpiresIn:    expiresIn,
			TokenVersion: user.TokenVersion,
		},
		RefreshTokenPlain: newPlain,
		CsrfTokenPlain:    csrfPlain,
	}, nil
}

func (u *AuthUsecase) Logout(ctx context.Context, refreshTokenPlain string) (*SuccessResponse, error) {
	if err := u.validator.ValidateLogout(ctx); err != nil {
		return nil, err
	}

	if refreshTokenPlain == "" {
		return nil, ErrUnauthorized
	}

	tokenHash := hashToken(refreshTokenPlain)

	rt, err := u.rtRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil || rt == nil {
		return nil, ErrUnauthorized
	}

	//refreshを削除（失効）
	if err := u.rtRepo.DeleteByID(ctx, rt.ID); err != nil {
		return nil, ErrInternal
	}

	return &SuccessResponse{Message: "logout success"}, nil
}

func (u *AuthUsecase) ForceLogout(ctx context.Context, targetUserID int64) (*ForceLogoutResponse, error) {
	if err := u.validator.ValidateForceLogout(ctx, targetUserID); err != nil {
		return nil, err
	}

	if targetUserID <= 0 {
		return nil, ErrValidation
	}

	if err := u.users.IncrementTokenVersion(ctx, targetUserID); err != nil {
		return nil, ErrInternal
	}

	if err := u.rtRepo.DeleteAllByUserID(ctx, targetUserID); err != nil {
		return nil, ErrInternal
	}

	//更新後を取得してnew_token_versionを返す
	user, err := u.users.FindByID(ctx, targetUserID)
	if err != nil || user == nil {
		return nil, ErrInternal
	}

	return &ForceLogoutResponse{
		UserID:          user.ID,
		NewTokenVersion: user.TokenVersion,
	}, nil
}

// jwt発行
func (u *AuthUsecase) issueAccessToken(user *model.User) (string, int, error) {
	now := time.Now()
	exp := now.Add(accessTokenTTL)

	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": string(user.Role),
		"tv":   user.TokenVersion,
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := t.SignedString([]byte(u.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return signed, int(accessTokenTTL.Seconds()), nil
}

// refresh token生成（平文 + DB保存hash）
func newRandomTokenAndHash() (plain string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}

	plain = base64.RawURLEncoding.EncodeToString(b)

	sum := sha256.Sum256([]byte(plain))
	hash = base64.RawURLEncoding.EncodeToString(sum[:])

	return plain, hash, nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func toUserDTOOpenAPI(u *model.User) (UserDTO, error) {

	return UserDTO{
		ID:           u.ID,
		Email:        u.Email,
		Role:         string(u.Role),
		TokenVersion: u.TokenVersion,
		IsActive:     u.IsActive,
	}, nil
}
