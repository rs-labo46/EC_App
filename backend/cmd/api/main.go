package main

import (
	"os"
	"time"

	"app/internal/domain/model"
	"app/internal/handler"
	"app/internal/infra/db"
	infraRepo "app/internal/infra/repository"
	"app/internal/server"
	auth "app/internal/usecase/auth_usecase"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type uuidGenerator struct{}

func (g *uuidGenerator) NewID() string {
	return uuid.NewString()
}

type realClock struct{}

func (c *realClock) Now() time.Time {
	return time.Now()
}

type jwtIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

func newJWTIssuerFromEnv() *jwtIssuer {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev_secret_change_me"
	}

	//アクセストークン
	return &jwtIssuer{
		secret:    []byte(secret),
		accessTTL: 15 * time.Minute,
	}
}

func (i *jwtIssuer) Issue(userID string, role model.Role, tokenVersion int, now time.Time) (string, time.Time, error) {
	expiresAt := now.Add(i.accessTTL)

	claims := jwt.MapClaims{
		"sub":           userID,
		"role":          string(role),
		"token_version": tokenVersion,
		"iat":           now.Unix(),
		"exp":           expiresAt.Unix(),
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

func main() {
	//DB接続
	if err := godotenv.Load("../.env"); err != nil {
		panic(err)
	}

	gormDB, _ := db.Connect()
	if err := gormDB.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
	); err != nil {
		panic(err)
	}

	//Repository（GORM実装）生成

	userRepo := infraRepo.NewUserRepository(gormDB)
	rtRepo := infraRepo.NewRefreshTokenRepository(gormDB)

	//usecaseに渡す部品
	idGen := &uuidGenerator{}
	clock := &realClock{}

	//bcrypt（会員登録：Hash / ログイン：Verify）
	hasher := auth.NewBcryptPasswordHasher(12)
	verifier := auth.NewBcryptPasswordVerifier()

	//JWT issuer
	issuer := newJWTIssuerFromEnv()

	//refresh TTL
	refreshTTL := 14 * 24 * time.Hour

	//Usecase生成
	registerUC := auth.NewRegisterUserUsecase(userRepo, hasher, idGen, clock)
	loginUC := auth.NewLoginUsecase(userRepo, rtRepo, verifier, issuer, idGen, clock, refreshTTL)

	//Handler生成
	authH := handler.NewAuthHandler(registerUC, loginUC, refreshTTL)

	//Server起動
	addr := ":8080"
	if v := os.Getenv("PORT"); v != "" {

		if v[0] != ':' {
			addr = ":" + v
		} else {
			addr = v
		}
	}

	if err := server.Start(addr, authH); err != nil {
		panic(err)
	}
}
