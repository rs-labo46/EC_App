package usecase

import (
	"context"
	"errors"
	"time"

	"app/internal/domain/model"
	"app/internal/repository"

	"gorm.io/gorm"
)

// 住所系で存在しないことを表す（Handlerが404に変換する）
var ErrNotFound = errors.New("not found")

type AddressDTO struct {
	ID         int64   `json:"id"`
	UserID     int64   `json:"user_id"`
	PostalCode string  `json:"postal_code"`
	Prefecture string  `json:"prefecture"`
	City       string  `json:"city"`
	Line1      string  `json:"line1"`
	Line2      string  `json:"line2"`
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	IsDefault  bool    `json:"is_default"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  *string `json:"updated_at,omitempty"`
}

type AddressCreateRequest struct {
	PostalCode string `json:"postal_code"`
	Prefecture string `json:"prefecture"`
	City       string `json:"city"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
}

type AddressUpdateRequest struct {
	PostalCode string `json:"postal_code"`
	Prefecture string `json:"prefecture"`
	City       string `json:"city"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	Name       string `json:"name"`
	Phone      string `json:"phone"`
}

type AddressUsecase struct {
	addresses repository.AddressRepository
}

func NewAddressUsecase(addresses repository.AddressRepository) *AddressUsecase {
	return &AddressUsecase{addresses: addresses}
}

func (u *AddressUsecase) List(ctx context.Context, userID int64) ([]AddressDTO, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}

	list, err := u.addresses.ListByUserID(ctx, userID)
	if err != nil {
		return nil, ErrInternal
	}

	out := make([]AddressDTO, 0, len(list))
	for i := range list {
		out = append(out, toAddressDTO(&list[i]))
	}
	return out, nil
}

func (u *AddressUsecase) Create(ctx context.Context, userID int64, req AddressCreateRequest) (AddressDTO, error) {
	if userID <= 0 {
		return AddressDTO{}, ErrUnauthorized
	}

	//入力チェック
	if req.PostalCode == "" || req.Prefecture == "" || req.City == "" || req.Line1 == "" || req.Name == "" {
		return AddressDTO{}, ErrValidation
	}

	now := time.Now()

	a := model.Address{
		UserID:     userID,
		PostalCode: req.PostalCode,
		Prefecture: req.Prefecture,
		City:       req.City,
		Line1:      req.Line1,
		Line2:      req.Line2,
		Name:       req.Name,
		Phone:      req.Phone,
		IsDefault:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	created, err := u.addresses.Create(ctx, a)
	if err != nil {
		return AddressDTO{}, ErrInternal
	}

	return toAddressDTO(&created), nil
}

func (u *AddressUsecase) Update(ctx context.Context, userID int64, addressID int64, req AddressUpdateRequest) error {
	if userID <= 0 {
		return ErrUnauthorized
	}
	if addressID <= 0 {
		return ErrValidation
	}

	//所有チェック（本人のみ）
	owned, err := u.addresses.IsOwnedByUser(ctx, addressID, userID)
	if err != nil {
		//住所存在確認して404に
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}
	if !owned {
		return ErrForbidden
	}

	a := model.Address{
		ID:         addressID,
		PostalCode: req.PostalCode,
		Prefecture: req.Prefecture,
		City:       req.City,
		Line1:      req.Line1,
		Line2:      req.Line2,
		Name:       req.Name,
		Phone:      req.Phone,
		UpdatedAt:  time.Now(),
	}

	if err := u.addresses.Update(ctx, a); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}

	return nil
}

func (u *AddressUsecase) Delete(ctx context.Context, userID int64, addressID int64) error {
	if userID <= 0 {
		return ErrUnauthorized
	}
	if addressID <= 0 {
		return ErrValidation
	}

	owned, err := u.addresses.IsOwnedByUser(ctx, addressID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}
	if !owned {
		return ErrForbidden
	}

	if err := u.addresses.Delete(ctx, addressID); err != nil {
		//注文が参照中などで削除できない 409
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrConflict
	}

	return nil
}

func (u *AddressUsecase) SetDefault(ctx context.Context, userID int64, addressID int64) error {
	if userID <= 0 {
		return ErrUnauthorized
	}
	if addressID <= 0 {
		return ErrValidation
	}

	owned, err := u.addresses.IsOwnedByUser(ctx, addressID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}
	if !owned {
		return ErrForbidden
	}

	//user内でdefaultは1つ
	if err := u.addresses.SetDefault(ctx, userID, addressID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return ErrInternal
	}

	return nil
}

func toAddressDTO(a *model.Address) AddressDTO {
	dto := AddressDTO{
		ID:         a.ID,
		UserID:     a.UserID,
		PostalCode: a.PostalCode,
		Prefecture: a.Prefecture,
		City:       a.City,
		Line1:      a.Line1,
		Line2:      a.Line2,
		Name:       a.Name,
		Phone:      a.Phone,
		IsDefault:  a.IsDefault,
		CreatedAt:  a.CreatedAt.Format(time.RFC3339),
	}
	t := a.UpdatedAt.Format(time.RFC3339)
	dto.UpdatedAt = &t
	return dto
}
