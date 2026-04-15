package store

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

func (s *GormStore) SaveEpisodicMemory(ctx context.Context, rec *EpisodicMemoryRecord) error {
	return s.db.WithContext(ctx).Create(rec).Error
}

func (s *GormStore) ListEpisodicMemories(ctx context.Context, symbol string, limit int) ([]EpisodicMemoryRecord, error) {
	var records []EpisodicMemoryRecord
	q := s.db.WithContext(ctx).Where("symbol = ?", symbol).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (s *GormStore) FindEpisodicMemoryByPosition(ctx context.Context, positionID string) (EpisodicMemoryRecord, bool, error) {
	var rec EpisodicMemoryRecord
	err := s.db.WithContext(ctx).Where("position_id = ?", positionID).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rec, false, nil
		}
		return rec, false, err
	}
	return rec, true, nil
}

func (s *GormStore) DeleteEpisodicMemoriesOlderThan(ctx context.Context, symbol string, before time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Where("symbol = ? AND created_at < ?", symbol, before).Delete(&EpisodicMemoryRecord{})
	return result.RowsAffected, result.Error
}
