package main

import (
	"log"
	"net/http"

	"app/internal/config"
	"app/internal/domain/model"
	"app/internal/handler"
	"app/internal/infra/db"
	infrarepo "app/internal/infra/repository"
	"app/internal/middleware"
	"app/internal/usecase"
	"app/internal/validator"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
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
		&model.Order{},
		&model.OrderItem{},
		&model.Address{},
		&model.AuditLog{},
	); err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	// Echoサーバを起動する
	e := echo.New()

	// CORS（フロント http://localhost:3000 からのアクセスを許可）
	// Cookie(refresh)を使うので AllowCredentials を true にする
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			"X-CSRF-Token",
			"X-Idempotency-Key",
		},
		AllowCredentials: true,
	}))

	// preflight(OPTIONS)は認証などに通さず、即204を返す
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	})

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

	//Address
	addrRepo := infrarepo.NewAddressGormRepository(gormDB)
	addrUC := usecase.NewAddressUsecase(addrRepo)
	addrHandler := handler.NewAddressHandler(addrUC)
	authGroup := e.Group(
		"",
		middleware.AuthJWT(cfg),
		middleware.TokenVersionGuard(userRepo),
	)

	addrHandler.RegisterRoutes(authGroup)

	//Handler(強制ログアウト)
	adminUserH := handler.NewAdminUserHandler(cfg, userRepo, authUC)
	adminUserH.RegisterRoutes(e)

	//監査ログ
	auditRepo := infrarepo.NewAuditLogGormRepository(gormDB)

	// Products
	productRepo := infrarepo.NewProductGormRepository(gormDB)
	inventoryRepo := infrarepo.NewInventoryGormRepository(gormDB)
	productUC := usecase.NewProductUsecase(productRepo, inventoryRepo, auditRepo)

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

	// TxManager
	txManager := infrarepo.NewTxManagerGorm(gormDB)

	// Orders
	orderUC := usecase.NewOrderUsecase(txManager, addrRepo)
	orderH := handler.NewOrderHandler(orderUC)
	orderH.RegisterRoutes(e, cfg, userRepo)

	//AdminOrder一覧
	adminOrderUC := usecase.NewAdminOrderUsecase(txManager, auditRepo)
	adminOrderH := handler.NewAdminOrderHandler(adminOrderUC)
	adminOrderH.RegisterRoutes(e, cfg, userRepo)

	// サーバ起動
	log.Fatal(e.Start(":" + cfg.Port))

}
