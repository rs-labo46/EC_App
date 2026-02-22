package main

import (
	"log"

	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/handler"
	"app/internal/infra/db"
	infrarepo "app/internal/infra/repository"
	"app/internal/usecase"
	"app/internal/validator"

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

	if err := gormDB.AutoMigrate(
		&model.User{},
		&model.RefreshToken{},
		&model.Product{},
		&model.InventoryAdjustment{},
		&model.Cart{},
		&model.CartItem{},
	); err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	// Echoサーバを起動する
	e := echo.New()

	// 疎通確認用エンドポイント
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "ok")
	})

	// DI（依存注入）
	// Repository（GORM実装）
	userRepo := infrarepo.NewUserGormRepository(gormDB)
	rtRepo := infrarepo.NewRefreshTokenGormRepository(gormDB)

	//Validator（usecase.AuthValidator の実装）
	authValidator := validator.NewAuthValidator(userRepo)

	//Usecase
	authUC := usecase.NewAuthUsecase(cfg, userRepo, rtRepo, authValidator)

	//Handler（ルーティング登録）
	authH := handler.NewAuthHandler(cfg, authUC, userRepo)
	authH.RegisterRoutes(e)

	//Handler(強制ログアウト)
	adminUserH := handler.NewAdminUserHandler(cfg, userRepo, authUC)
	adminUserH.RegisterRoutes(e)

	// Products
	productRepo := infrarepo.NewProductGormRepository(gormDB)
	inventoryRepo := infrarepo.NewInventoryGormRepository(gormDB)
	productUC := usecase.NewProductUsecase(productRepo, inventoryRepo)

	productH := handler.NewProductHandler(productUC)
	productH.RegisterRoutes(e)

	adminProductH := handler.NewAdminProductHandler(productUC)
	adminProductH.RegisterRoutes(e, cfg, userRepo)

	// Cart
	cartRepoImpl := infrarepo.NewCartGormRepository(gormDB)
	// usecase
	cartUC := usecase.NewCartUsecase(cartRepoImpl, cartRepoImpl, productRepo)
	cartH := handler.NewCartHandler(cartUC)
	cartH.RegisterRoutes(e, cfg, userRepo)

	// サーバ起動
	log.Fatal(e.Start(":" + cfg.Port))

}
