ECプロジェクトの **バックエンド** と **フロントエンド** をまとめたものです。  
仕様書（OpenAPI要件）に合わせて、**認証 / 商品 / 在庫 / カート / 注文 / 管理者操作 / 住所 / 監査ログ** を実装しています。

# EC Backend (Go + Echo + PostgreSQL + GORM)

## 技術スタック

- Go
- Echo（HTTPサーバ）
- PostgreSQL（Docker Compose）
- GORM（ORM）
- JWT（Bearer認証）
- Refresh Token（回転 + 再利用検知）
- CSRF（Double Submit Cookie / refreshとlogoutで使用）
- E2Eテスト（Go test）

---

## できること（機能一覧）

### 認証（Auth）

- Register（ユーザー登録）
- Login（access token発行 + refresh cookie set）
- Refresh（refresh回転 + CSRF必須 + 再利用検知）
- Logout（CSRF必須 + bearer必須）
- /me（bearer必須 + token_version一致必須）
- Force Logout（admin only / token_version++ による既存JWT無効化）

### 商品（Products）/ 在庫（Inventory）

- 公開商品一覧/詳細（公開のみ、検索・ページング・ソート）
- 管理者 CRUD（admin only、論理削除）
- 在庫更新（admin only、履歴 inventory_adjustments に記録）
- 監査ログ（在庫更新時に AuditLog を記録）

### カート（Cart）

- ACTIVEカートはユーザーにつき最大1
- 追加（同一商品は数量加算）
- 更新（cart_item.id）
- 削除

### 注文（Orders）

- 注文作成（Tx + idempotency_key 二重送信防止 + 在庫減算 + カートクリア）
- address_id 必須 + 所有チェック
- 注文一覧/詳細（本人のみ）

### 管理者注文（Admin Orders）

- 注文一覧（admin only）
- 注文ステータス更新（admin only）
- 状態遷移ガード（終端：SHIPPED/CANCELEDは変更禁止、SHIPPED→CANCELED禁止）
- CANCELEDへの遷移で在庫戻し（PENDING/PAIDのみ）
- 監査ログ（注文ステータス更新時に AuditLog を記録）

---

## ディレクトリ構成

- cmd/api/main.go # エントリポイント（DI, AutoMigrate, ルート登録）
- internal/config # env読み込み
- internal/domain/model # Entity（User/Product/Cart/Order/Address/AuditLog...）
- internal/repository # interface（usecaseが依存する）
- internal/infra/db # DB接続（GORM）
- internal/infra/repository # GORM実装（interfaceの外側実装）
- internal/usecase # ビジネスロジック（Tx・所有チェック・状態遷移ガード等）
- internal/handler # HTTP（入力/出力・ステータス変換・ルーティング）
- internal/middleware # AuthJWT / TokenVersionGuard / AdminRoleGuard / CSRF
- tests/e2e # E2Eテスト
- tests/unit # 単体テスト

### クリーンアーキテクチャの依存関係

- **usecase は repository(interface) にだけ依存**します
- GORMの実装は **infra** に置き、usecaseからは見えないようにします
- handler は usecase を呼びます

---

## 起動方法

EC_App/ec ディレクトリで実行です。
docker compose up --build

## E2E テスト

サーバを起動したまま、別ターミナルで：
cd backend

- go clean -testcache
- go test ./tests/e2e -v
- go test ./tests/unit -v

## Register（ユーザー登録）

※Registerはユーザー作成のみです。**access token / refresh cookie / csrf cookie は発行されません**。  
続けて `/auth/login` を実行してログインしてください。

curl -i -X POST http://localhost:8080/auth/register \
 -H "Content-Type: application/json" \
 -d '{"email":"user1@test.com","password":"CorrectPW123!"}'

## Login（access token発行 + refresh cookie + csrf cookie）

curl -i -c cookies.txt -X POST http://localhost:8080/auth/login \
 -H "Content-Type: application/json" \
 -d '{"email":"user1@test.com","password":"CorrectPW123!"}'

# loginレスポンス(JSON)の token.access_token を $ACCESS に入れる

# 例: ACCESS="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

## Me（bearer必須）

curl -i http://localhost:8080/me \
 -H "Authorization: Bearer $ACCESS"

## Refresh（CSRF必須 + refresh回転）

CSRF=$(grep csrf_token cookies.txt | awk '{print $7}')
curl -i -b cookies.txt -c cookies.txt -X POST http://localhost:8080/auth/refresh \
 -H "X-CSRF-Token: $CSRF"

## Logout（CSRF必須 + bearer必須）

CSRF=$(grep csrf_token cookies.txt | awk '{print $7}')
curl -i -b cookies.txt -c cookies.txt -X POST http://localhost:8080/auth/logout \
 -H "Authorization: Bearer $ACCESS" \
 -H "X-CSRF-Token: $CSRF"

## Address（住所）

- 住所作成（bearer必須）
  curl -i -X POST http://localhost:8080/addresses \
   -H "Authorization: Bearer $ACCESS" \
   -H "Content-Type: application/json" \
   -d '{
  "postal_code":"5300001",
  "prefecture":"大阪府",
  "city":"大阪市北区",
  "line1":"梅田1-1-1",
  "line2":"",
  "name":"山田太郎",
  "phone":"09000000000"
  }'

- 一覧
  curl -i http://localhost:8080/addresses \
   -H "Authorization: Bearer $ACCESS"

- default切替
  curl -i -X POST http://localhost:8080/addresses/1/default \
   -H "Authorization: Bearer $ACCESS"

## Orders（注文）

- 注文作成（address_id必須 + 二重送信防止ヘッダー必須）
  curl -i -X POST http://localhost:8080/orders \
   -H "Authorization: Bearer $ACCESS" \
   -H "Content-Type: application/json" \
   -H "X-Idempotency-Key: order-key-001" \
   -d '{"address_id":1}'
- 注文一覧（本人のみ）
  curl -i http://localhost:8080/orders \
   -H "Authorization: Bearer $ACCESS"

## Admin（管理者）

- 在庫更新（admin only）※監査ログが残る
  curl -i -X PUT http://localhost:8080/admin/inventory/1 \
   -H "Authorization: Bearer $ACCESS" \
   -H "Content-Type: application/json" \
   -d '{"stock":10,"reason":"manual adjust"}'
- 注文ステータス更新（admin only）※監査ログが残る
  curl -i -X PUT http://localhost:8080/admin/orders/1/status \
   -H "Authorization: Bearer $ACCESS" \
   -H "Content-Type: application/json" \
   -d '{"status":"SHIPPED"}'

# EC Frontend (React + Vite + TypeScript)

## 技術スタック

- React
- Vite
- TypeScript
- Fetch
- Docker Compose

---

## できること（機能一覧）

### 認証（Auth）

- Login（access token取得 + refresh cookie set + csrf cookie set）
- Refresh（refresh回転 + CSRF必須 + 再利用検知）
- Logout（CSRF必須 + refresh失効）
- /me（bearer必須 + token_version一致必須）※backend側実装に合わせる
- Force Logout（admin only / token_version++ による既存JWT無効化）

### 商品（Products）

- 公開商品一覧（検索・ページング・ソート）
- 公開商品詳細

### カート（Cart）※ログイン必須

- カート取得
- 追加（同一商品は数量加算 / 在庫超過は禁止）
- 数量変更
- 削除

### 注文（Orders）※ログイン必須

- 注文確定（address_id 必須 + Idempotency-Key 必須）
- 注文一覧/詳細（本人のみ）

### 住所（Addresses）※ログイン必須

- 住所作成 / 一覧 / 更新 / 削除 / default切替

### 管理者（Admin）※ADMIN必須

- 商品CRUD / 公開切替
- 在庫調整（理由必須、履歴あり）
- 注文一覧/ステータス更新（遷移ルールあり）
- ユーザー一覧/権限/停止/強制ログアウト

---

## ディレクトリ構成

- src/
  - pages/
    - Address.tsx
    - AdminProductCreatePage.tsx
    - Cart.tsx
    - Checkout.tsx
    - Signup.tsx
    - Login.tsx
    - Order.tsx
    - OrderDetail.tsx
    - ProductDetail.tsx
    - Product.tsx
  - ui/
    - styles.ts
  - components/
    - NavBar.tsx
  - api/
  - auth/
  - types/
  - api.ts
  - auth.ts
  - App.tsx

---

## 起動方法

EC_App/ec ディレクトリで実行です。
docker compose up --build
