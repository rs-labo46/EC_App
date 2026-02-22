package repository

import (
	"app/internal/domain/model"
	"context"
)

// 住所(Address)を保存・取得する窓口
type AddressRepository interface {
	//Create は住所を新規作成する。
	//作成後はaddress（IDなどが埋まったもの）を返す
	Create(ctx context.Context, address model.Address) (model.Address, error)

	//ユーザーが持つ住所一覧を返す
	ListByUserID(ctx context.Context, userID int64) ([]model.Address, error)

	//住所IDから住所を1件取得
	FindByID(ctx context.Context, addressID int64) (model.Address, error)

	//住所の更新。
	Update(ctx context.Context, address model.Address) error

	//住所の削除。
	Delete(ctx context.Context, addressID int64) error

	//住所がそのユーザーのものか」を確認
	IsOwnedByUser(ctx context.Context, addressID, userID int64) (bool, error)

	//住所の切り替えを行う。
	SetDefault(ctx context.Context, userID, addressID int64) error
}
