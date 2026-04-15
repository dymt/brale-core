package store

import (
	"context"
	"fmt"

	"brale-core/internal/decision/decisionutil"
)

func (s *GormStore) ListProviderEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]ProviderEventRecord, error) {
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	var out []ProviderEventRecord
	if err := s.db.WithContext(ctx).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, start, end).
		Order("timestamp asc, role asc").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *GormStore) ListAgentEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]AgentEventRecord, error) {
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	var out []AgentEventRecord
	if err := s.db.WithContext(ctx).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, start, end).
		Order("timestamp asc, stage asc").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *GormStore) ListGateEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]GateEventRecord, error) {
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	var out []GateEventRecord
	if err := s.db.WithContext(ctx).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, start, end).
		Order("timestamp asc").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *GormStore) ListDistinctSnapshotIDs(ctx context.Context, symbol string, start, end int64) ([]uint, error) {
	symbol = decisionutil.NormalizeSymbol(symbol)
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	var ids []uint
	if err := s.db.WithContext(ctx).
		Model(&GateEventRecord{}).
		Where("symbol = ? AND timestamp >= ? AND timestamp <= ?", symbol, start, end).
		Distinct("snapshot_id").
		Order("snapshot_id asc").
		Pluck("snapshot_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
