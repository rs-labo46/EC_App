package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"app/internal/domain/model"
	"app/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// 会員登録の入力
type RegisterUserInput struct {
	Email    string
	Password string
}

// 会員登録の出力
type RegisterUserOutput struct {
	User model.User
}

// bcryptハッシュ化
type BcryptPasswordHasher struct {
	cost int
}

var (
	// 入力が不正
	ErrInvalidEmailFormat = errors.New("invalid email format")
	ErrPasswordTooShort   = errors.New("password too short")
	ErrWeakPassword       = errors.New("weak password")

	// 競合
	ErrEmailAlreadyExists = errors.New("email already exists")
)

// 平文パスワードからハッシュへ。
type PasswordHasher interface {
	Hash(plain string) (string, error)
}

// UUID 等のIDを作る約束
type IDGenerator interface {
	NewID() string
}

// 現在の時間
type Clock interface {
	Now() time.Time
}

// RegisterUserUsecaseは会員登録の処理。
type RegisterUserUsecase struct {
	userRepo repository.UserRepository
	hasher   PasswordHasher
	idGen    IDGenerator
	clock    Clock
}

// DI
func NewRegisterUserUsecase(
	userRepo repository.UserRepository,
	hasher PasswordHasher,
	idGen IDGenerator,
	clock Clock,
) *RegisterUserUsecase {
	return &RegisterUserUsecase{
		userRepo: userRepo,
		hasher:   hasher,
		idGen:    idGen,
		clock:    clock,
	}
}

// 会員登録実行
func (u *RegisterUserUsecase) Execute(ctx context.Context, in RegisterUserInput) (RegisterUserOutput, error) {
	var out RegisterUserOutput

	// emailの形式チェック
	if !isValidEmailFormat(in.Email) {
		return out, ErrInvalidEmailFormat
	}

	// password の長さチェック（最小12文字）
	if len(in.Password) < 12 {
		return out, ErrPasswordTooShort
	}

	// よくある弱いパスワードの拒否
	if isWeakPassword(in.Password) {
		return out, ErrWeakPassword
	}

	// email重複チェック
	existing, err := u.userRepo.FindByEmail(ctx, in.Email)
	if err == nil && existing != nil {
		return out, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		return out, err
	}

	// パスワードをハッシュ化
	hashed, err := u.hasher.Hash(in.Password)
	if err != nil {
		return out, err
	}

	// Userを作って保存
	now := u.clock.Now()

	user := &model.User{
		ID:           u.idGen.NewID(),
		Email:        in.Email,
		Password:     hashed,         // ハッシュを保存（平文は保存しない）
		Role:         model.RoleUser, // 初期はUSER
		TokenVersion: 0,
		IsActive:     true,
		LastLoginAt:  nil,
		Timestamps: model.Timestamp{
			CreatedAt: now,
			UpdatedAt: now,
		},
	} // ← ★ここで構造体リテラルを閉じる

	// DBへ保存
	if err := u.userRepo.Create(ctx, user); err != nil {
		return out, err
	}

	// 返すときは password を空にして漏洩防止
	safeUser := *user
	safeUser.Password = ""

	out.User = safeUser
	return out, nil
}

// メールチェック
func isValidEmailFormat(email string) bool {
	trimmed := strings.TrimSpace(email)
	if trimmed == "" {
		return false
	}
	_, err := mail.ParseAddress(trimmed)
	return err == nil
}

// パスワードのよくある弱いパスワード
func isWeakPassword(password string) bool {
	normalized := strings.ToLower(strings.TrimSpace(password))

	weak := map[string]struct{}{
		"password":     {},
		"password123":  {},
		"123456789012": {},
		"1234567890":   {},
		"12345678":     {},
		"qwerty":       {},
		"qwertyuiop":   {},
		"letmein":      {},
		"admin":        {},
		"admin123":     {},
	}

	_, ok := weak[normalized]
	return ok
}

// DI
func NewBcryptPasswordHasher(cost int) *BcryptPasswordHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptPasswordHasher{cost}
}

// bcryptでハッシュ化
func (h *BcryptPasswordHasher) Hash(plain string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// bcryptハッシュと平文を比較
type BcryptPasswordVerifier struct{}

// DI
func NewBcryptPasswordVerifier() *BcryptPasswordVerifier {
	return &BcryptPasswordVerifier{}
}

// 平文(plain)をbcryptで比較
func (v *BcryptPasswordVerifier) Verify(plain string, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}
