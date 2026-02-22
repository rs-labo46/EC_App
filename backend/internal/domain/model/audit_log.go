package model

import "time"

// 在庫更新、注文ステータス更新など。
type AuditAction string

const (
	//在庫を更新した操作。
	AuditActionUpdateStock AuditAction = "UPDATE_STOCK"
	//注文ステータスを更新した操作。
	AuditActionUpdateOrderStatus AuditAction = "UPDATE_ORDER_STATUS"
)

// 何に対する操作か
type AuditResourceType string

const (
	//商品に対する操作。
	AuditResourceProduct AuditResourceType = "product"

	//注文に対する操作。
	AuditResourceOrder AuditResourceType = "order"

	//ユーザーに対する操作。
	AuditResourceUser AuditResourceType = "user"
)

// 監査ログ（管理者操作ログ）。
// 「誰が」「何を」「どの対象に」「どう変えたか」を残す。
type AuditLog struct {
	//IDは監査ログの主キー
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`

	//操作したユーザー（主に管理者）のID。
	ActorUserID int64 `gorm:"not null;index" json:"actor_user_id"`

	//Actionは操作の種類（UPDATE_STOCK / UPDATE_ORDER_STATUS など）。
	Action AuditAction `gorm:"type:varchar(50);not null;index" json:"action"`

	//対象の種類（product / order / user）。
	ResourceType AuditResourceType `gorm:"type:varchar(50);not null;index" json:"resource_type"`

	//対象のID）。
	ResourceID int64 `gorm:"not null;index" json:"resource_id"`

	//JSON文字列で保存する。
	BeforeJSON string `gorm:"type:text" json:"before_json"`

	//JSON文字列で保存する。
	AfterJSON string `gorm:"type:text" json:"after_json"`

	//作成時刻
	CreatedAt time.Time `gorm:"not null;index" json:"created_at"`
}
