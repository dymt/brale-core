package store

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func (s *GormStore) SaveSemanticMemory(ctx context.Context, rec *SemanticMemoryRecord) error {
	return s.db.WithContext(ctx).Create(rec).Error
}

func (s *GormStore) UpdateSemanticMemory(ctx context.Context, id uint, updates map[string]any) error {
	return s.db.WithContext(ctx).Model(&SemanticMemoryRecord{}).Where("id = ?", id).Updates(updates).Error
}

func (s *GormStore) DeleteSemanticMemory(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&SemanticMemoryRecord{}, id).Error
}

func (s *GormStore) ListSemanticMemories(ctx context.Context, symbol string, activeOnly bool, limit int) ([]SemanticMemoryRecord, error) {
	var records []SemanticMemoryRecord
	q := s.db.WithContext(ctx)
	if symbol != "" {
		q = q.Where("symbol = ? OR symbol = ''", symbol)
	}
	if activeOnly {
		q = q.Where("active = ?", true)
	}
	q = q.Order("confidence DESC, created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (s *GormStore) FindSemanticMemory(ctx context.Context, id uint) (SemanticMemoryRecord, bool, error) {
	var rec SemanticMemoryRecord
	err := s.db.WithContext(ctx).First(&rec, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rec, false, nil
		}
		return rec, false, err
	}
	return rec, true, nil
}
