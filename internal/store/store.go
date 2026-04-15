package store

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type EventCommandStore interface {
	SaveAgentEvent(ctx context.Context, rec *AgentEventRecord) error
	SaveProviderEvent(ctx context.Context, rec *ProviderEventRecord) error
	SaveGateEvent(ctx context.Context, rec *GateEventRecord) error
	SaveRiskPlanHistory(ctx context.Context, rec *RiskPlanHistoryRecord) error
}

type PositionCommandStore interface {
	SavePosition(ctx context.Context, rec *PositionRecord) error
	UpdatePosition(ctx context.Context, positionID string, expectedVersion int, updates map[string]any) (bool, error)
	UpdatePositionPatch(ctx context.Context, patch PositionPatch) (bool, error)
}

type PositionQueryStore interface {
	FindPositionByID(ctx context.Context, positionID string) (PositionRecord, bool, error)
	FindPositionBySymbol(ctx context.Context, symbol string, statuses []string) (PositionRecord, bool, error)
	ListPositionsByStatus(ctx context.Context, statuses []string) ([]PositionRecord, error)
}

type RiskPlanQueryStore interface {
	FindLatestRiskPlanHistory(ctx context.Context, positionID string) (RiskPlanHistoryRecord, bool, error)
	ListRiskPlanHistory(ctx context.Context, positionID string, limit int) ([]RiskPlanHistoryRecord, error)
}

type TimelineQueryStore interface {
	ListProviderEvents(ctx context.Context, symbol string, limit int) ([]ProviderEventRecord, error)
	ListProviderEventsBySnapshot(ctx context.Context, symbol string, snapshotID uint) ([]ProviderEventRecord, error)
	ListProviderEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]ProviderEventRecord, error)
	ListAgentEvents(ctx context.Context, symbol string, limit int) ([]AgentEventRecord, error)
	ListAgentEventsBySnapshot(ctx context.Context, symbol string, snapshotID uint) ([]AgentEventRecord, error)
	ListAgentEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]AgentEventRecord, error)
	ListGateEvents(ctx context.Context, symbol string, limit int) ([]GateEventRecord, error)
	ListGateEventsByTimeRange(ctx context.Context, symbol string, start, end int64) ([]GateEventRecord, error)
	FindGateEventBySnapshot(ctx context.Context, symbol string, snapshotID uint) (GateEventRecord, bool, error)
	ListDistinctSnapshotIDs(ctx context.Context, symbol string, start, end int64) ([]uint, error)
}

type SymbolCatalogQueryStore interface {
	ListSymbols(ctx context.Context) ([]string, error)
}

type EpisodicMemoryStore interface {
	SaveEpisodicMemory(ctx context.Context, rec *EpisodicMemoryRecord) error
	ListEpisodicMemories(ctx context.Context, symbol string, limit int) ([]EpisodicMemoryRecord, error)
	FindEpisodicMemoryByPosition(ctx context.Context, positionID string) (EpisodicMemoryRecord, bool, error)
	DeleteEpisodicMemoriesOlderThan(ctx context.Context, symbol string, before time.Time) (int64, error)
}

type SemanticMemoryStore interface {
	SaveSemanticMemory(ctx context.Context, rec *SemanticMemoryRecord) error
	UpdateSemanticMemory(ctx context.Context, id uint, updates map[string]any) error
	DeleteSemanticMemory(ctx context.Context, id uint) error
	ListSemanticMemories(ctx context.Context, symbol string, activeOnly bool, limit int) ([]SemanticMemoryRecord, error)
	FindSemanticMemory(ctx context.Context, id uint) (SemanticMemoryRecord, bool, error)
}

type Store interface {
	EventCommandStore
	PositionCommandStore
	PositionQueryStore
	RiskPlanQueryStore
	TimelineQueryStore
	SymbolCatalogQueryStore
	EpisodicMemoryStore
	SemanticMemoryStore
}

type GormStore struct {
	db *gorm.DB
}

var _ EventCommandStore = (*GormStore)(nil)
var _ PositionCommandStore = (*GormStore)(nil)
var _ PositionQueryStore = (*GormStore)(nil)
var _ RiskPlanQueryStore = (*GormStore)(nil)
var _ TimelineQueryStore = (*GormStore)(nil)
var _ SymbolCatalogQueryStore = (*GormStore)(nil)
var _ EpisodicMemoryStore = (*GormStore)(nil)
var _ SemanticMemoryStore = (*GormStore)(nil)
var _ Store = (*GormStore)(nil)

func NewStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}
