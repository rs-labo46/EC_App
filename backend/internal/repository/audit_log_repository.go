package repository

import (
	"context"
	"time"

	"app/internal/domain/model"
)

//監査ログの絞り込み条件。

type AuditLogFilter struct {
	ActorUserID  *int64
	Action       *model.AuditAction
	ResourceType *model.AuditResourceType
	ResourceID   *int64
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	Limit        int
	Offset       int
}

// 監査ログの保存・一覧取得の約束。
type AuditLogRepository interface {
	//監査ログを1件保存
	Create(ctx context.Context, log model.AuditLog) error

	//監査ログを条件で一覧取得。
	List(ctx context.Context, filter AuditLogFilter) ([]model.AuditLog, error)
}
