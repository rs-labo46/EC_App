package repository

import (
	"context"

	"app/internal/domain/model"
	repo "app/internal/repository"

	"gorm.io/gorm"
)

type auditLogGormRepository struct {
	db *gorm.DB
}

func NewAuditLogGormRepository(db *gorm.DB) repo.AuditLogRepository {
	return &auditLogGormRepository{db: db}
}

func (r *auditLogGormRepository) Create(ctx context.Context, log model.AuditLog) error {
	if err := r.db.WithContext(ctx).Create(&log).Error; err != nil {
		return err
	}
	return nil
}

func (r *auditLogGormRepository) List(ctx context.Context, filter repo.AuditLogFilter) ([]model.AuditLog, error) {
	q := r.db.WithContext(ctx).Model(&model.AuditLog{})

	if filter.ActorUserID != nil {
		q = q.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}
	if filter.ResourceType != nil {
		q = q.Where("resource_type = ?", *filter.ResourceType)
	}
	if filter.ResourceID != nil {
		q = q.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.CreatedFrom != nil {
		q = q.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		q = q.Where("created_at <= ?", *filter.CreatedTo)
	}

	//新しい順
	q = q.Order("id DESC")

	// limit/offset
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	q = q.Limit(limit).Offset(filter.Offset)

	var logs []model.AuditLog
	if err := q.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
