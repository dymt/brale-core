package store

import (
	"fmt"
	"strings"
)

type PositionPatch struct {
	PositionID         string
	ExpectedVersion    int
	NextVersion        int
	Status             *string
	CloseIntentID      *string
	CloseSubmittedAt   *int64
	ExecutorPositionID *string
	Qty                *float64
	AvgEntry           *float64
	InitialStake       *float64
	Source             *string
	AbortReason        *string
	AbortStartedAt     *int64
	AbortFinalizedAt   *int64
	RiskJSON           *[]byte
}

func (p PositionPatch) Validate() error {
	if strings.TrimSpace(p.PositionID) == "" {
		return fmt.Errorf("position_id is required")
	}
	if p.ExpectedVersion <= 0 {
		return fmt.Errorf("expected_version must be > 0")
	}
	if p.NextVersion != p.ExpectedVersion+1 {
		return fmt.Errorf("next_version must equal expected_version + 1")
	}
	if len(p.Updates()) == 0 {
		return fmt.Errorf("position patch has no updates")
	}
	return nil
}

func (p PositionPatch) Updates() map[string]any {
	updates := map[string]any{}
	if p.Status != nil {
		updates["status"] = *p.Status
	}
	if p.CloseIntentID != nil {
		updates["close_intent_id"] = *p.CloseIntentID
	}
	if p.CloseSubmittedAt != nil {
		updates["close_submitted_at"] = *p.CloseSubmittedAt
	}
	if p.ExecutorPositionID != nil {
		updates["executor_position_id"] = *p.ExecutorPositionID
	}
	if p.Qty != nil {
		updates["qty"] = *p.Qty
	}
	if p.AvgEntry != nil {
		updates["avg_entry"] = *p.AvgEntry
	}
	if p.InitialStake != nil {
		updates["initial_stake"] = *p.InitialStake
	}
	if p.Source != nil {
		updates["source"] = *p.Source
	}
	if p.AbortReason != nil {
		updates["abort_reason"] = *p.AbortReason
	}
	if p.AbortStartedAt != nil {
		updates["abort_started_at"] = *p.AbortStartedAt
	}
	if p.AbortFinalizedAt != nil {
		updates["abort_finalized_at"] = *p.AbortFinalizedAt
	}
	if p.RiskJSON != nil {
		updates["risk_json"] = *p.RiskJSON
	}
	return updates
}

func PtrString(v string) *string    { return &v }
func PtrInt64(v int64) *int64       { return &v }
func PtrFloat64(v float64) *float64 { return &v }
func PtrBytes(v []byte) *[]byte     { return &v }
