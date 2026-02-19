package db

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect はDBに接続して *gorm.DB を返す。
func Connect() (*gorm.DB, error) {
	// DATABASE_URL があれば最優先で使う
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})
	}

	host := getenv("POSTGRES_HOST", "localhost")
	port := getenv("POSTGRES_PORT", "5432")
	user := getenv("POSTGRES_USER", "postgres")
	pass := getenv("POSTGRES_PASSWORD", "postgres")
	name := getenv("POSTGRES_DB", "app")
	ssl := getenv("POSTGRES_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, ssl,
	)

	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func getenv(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
