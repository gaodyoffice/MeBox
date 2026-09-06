package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/truewhile/MeBox/internal/model"
)

// CloudOrganizeHistoryRepository persists model.CloudOrganizeHistory.
type CloudOrganizeHistoryRepository struct{ db *gorm.DB }

func (r *CloudOrganizeHistoryRepository) Create(ctx context.Context, h *model.CloudOrganizeHistory) error {
	return withSQLiteBusyRetry(ctx, func() error {
		return r.db.WithContext(ctx).Create(h).Error
	})
}

func (r *CloudOrganizeHistoryRepository) FindByID(ctx context.Context, id string) (*model.CloudOrganizeHistory, error) {
	var h model.CloudOrganizeHistory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&h).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *CloudOrganizeHistoryRepository) List(ctx context.Context, limit int) ([]model.CloudOrganizeHistory, error) {
	var rows []model.CloudOrganizeHistory
	q := r.db.WithContext(ctx).Order("created_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
	return rows, err
}

func (r *CloudOrganizeHistoryRepository) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).Where("created_at < ?", before).Delete(&model.CloudOrganizeHistory{})
	return result.RowsAffected, result.Error
}

func (r *CloudOrganizeHistoryRepository) DeleteAll(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Delete(&model.CloudOrganizeHistory{})
	return result.RowsAffected, result.Error
}
