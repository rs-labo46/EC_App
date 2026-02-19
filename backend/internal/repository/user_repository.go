package repository

import (
	"app/internal/domain/model"
	"context"
	"errors"
)

// ユーザーが見つかりませんを統一
var ErrUserNotFound = errors.New("user not found")

// 保存・取得を約束
type UserRepository interface {
	//新規ユーザー作成
	Create(ctx context.Context, user *model.User) error
	// IDからユーザーを1件取得する。
	FindByID(ctx context.Context, userID string) (*model.User, error)
	//メールからユーザーを一件取得する。
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	// ユーザー情報の更新=>アクティブかどうか・ロールの変更・最後のログイン更新など
	Update(ctx context.Context, user *model.User) error
	//トークンのバージョンを＋１
	IncrementTokenVersion(ctx context.Context, userID string) error
}
