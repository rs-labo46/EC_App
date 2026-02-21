package repository

import (
	"app/internal/domain/model"
	"context"
	"errors"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// リフレッシュトークンの保存・取得・更新・削除を行う約束。
type RefreshTokenRepository interface {
	//新しいリフレッシュトークンを保存。
	Create(ctx context.Context, token *model.RefreshToken) error

	//token_hash で検索。refreshのときにDBに存在するか・使われていないかを確認する。
	FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)

	//このトークンを使用済みにする処理（used_at をセット）
	MarkUsed(ctx context.Context, tokenID string) error

	//このトークンを無効化する処理revoked_at をセット）
	Revoke(ctx context.Context, tokenID string) error

	//そのユーザーのrefreshを全部消す処理です。再利用検知や強制ログアウトで必要。
	DeleteAllByUserID(ctx context.Context, userID int64) error

	//1件削除
	DeleteByID(ctx context.Context, tokenID string) error
}
