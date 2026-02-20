package config

import (
	"fmt"
	"os"
	"strconv"
)

// Configはアプリ全体の設定
type Config struct {
	Port string // サーバーポート（8080）

	PostgresUser     string // DBユーザー
	PostgresPassword string // DBパスワード
	PostgresDB       string // DB名
	PostgresHost     string // DBホスト（localhost）
	PostgresPort     int    // DBポート（5433）

	JWTSecret string // JWT署名シークレット

	GoEnv     string // dev/prod
	APIDomain string // APIドメイン（cookieやCORSなどで使う）
	FEURL     string // フロントURL（CORSなどで使う）
}

// Loadは環境変数
func Load() (Config, error) {
	pgPort, err := mustAtoi("POSTGRES_PORT")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port: os.Getenv("PORT"),

		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     pgPort,

		JWTSecret: os.Getenv("JWT_SECRET"),

		GoEnv:     os.Getenv("GO_ENV"),
		APIDomain: os.Getenv("API_DOMAIN"),
		FEURL:     os.Getenv("FE_URL"),
	}

	//必須チェック
	if cfg.Port == "" {
		return Config{}, fmt.Errorf("PORT is required")
	}
	if cfg.PostgresUser == "" {
		return Config{}, fmt.Errorf("POSTGRES_USER is required")
	}
	if cfg.PostgresPassword == "" {
		return Config{}, fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if cfg.PostgresDB == "" {
		return Config{}, fmt.Errorf("POSTGRES_DB is required")
	}
	if cfg.PostgresHost == "" {
		return Config{}, fmt.Errorf("POSTGRES_HOST is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.GoEnv == "" {
		return Config{}, fmt.Errorf("GO_ENV is required")
	}
	if cfg.APIDomain == "" {
		return Config{}, fmt.Errorf("API_DOMAIN is required")
	}
	if cfg.FEURL == "" {
		return Config{}, fmt.Errorf("FE_URL is required")
	}

	return cfg, nil
}

func mustAtoi(key string) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be number: %w", key, err)
	}
	return i, nil
}
