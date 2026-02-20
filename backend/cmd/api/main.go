package main

import (
	"log"

	"app/internal/config"
	"app/internal/infra/db"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func main() {
	// .env を読む
	_ = godotenv.Load()

	// 設定を読み込む（PORTやDB設定など）
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	// DBへ接続する（GORM）
	gormDB, err := db.NewGorm(cfg)
	if err != nil {
		log.Fatalf("db error: %v", err)
	}

	// DB接続できたことをログで確認する
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("db error: %v", err)
	}
	log.Println("db connected:", sqlDB.Stats().OpenConnections)

	// Echoサーバを起動する
	e := echo.New()

	// 疎通確認用エンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "ok")
	})

	// サーバ起動
	log.Fatal(e.Start(":" + cfg.Port))
}
