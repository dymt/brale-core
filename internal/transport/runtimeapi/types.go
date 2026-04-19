package runtimeapi

import (
	"time"

	"brale-core/internal/config"
	"brale-core/internal/decision/provider"
	"brale-core/internal/decision/decisionfmt"
	"brale-core/internal/execution"
	"brale-core/internal/runtime"
)

type errorResponse struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	RequestID string `json:"request_id"`
	Details   any    `json:"details,omitempty"`
}

type scheduleResponse struct {
	Status       string                  `json:"status"`
	LLMScheduled bool                    `json:"llm_scheduled"`
	Mode         string                  `json:"mode"`
	NextRuns     []runtime.SymbolNextRun `json:"next_runs,omitempty"`
	Positions    []PositionStatusItem    `json:"positions,omitempty"`
	Summary      string                  `json:"summary"`
	RequestID    string                  `json:"request_id"`
}

type MonitorStatusResponse struct {
	Status    string                `json:"status"`
	Symbols   []MonitorSymbolConfig `json:"symbols"`
	Summary   string                `json:"summary"`
	RequestID string                `json:"request_id"`
}

type MonitorRiskPlan struct {
	Mode         string                 `json:"mode"`
	Label        string                 `json:"label"`
	EntryPricing MonitorEntryPricing    `json:"entry_pricing"`
	Initial      MonitorRiskPlanSection `json:"initial"`
	Tighten      MonitorRiskPlanSection `json:"tighten"`
}

type MonitorEntryPricing struct {
	Mode  string `json:"mode"`
	Label string `json:"label"`
}

type MonitorRiskPlanSection struct {
	Source string         `json:"source"`
	Label  string         `json:"label"`
	Params map[string]any `json:"params,omitempty"`
}

type MonitorSymbolConfig struct {
	Symbol        string          `json:"symbol"`
	NextRun       time.Time       `json:"next_run"`
	KlineInterval string          `json:"kline_interval"`
	RiskPct       float64         `json:"risk_pct"`
	RiskAmount    float64         `json:"risk_amount"`
	MaxLeverage   float64         `json:"max_leverage"`
	RiskPlan      MonitorRiskPlan `json:"risk_plan"`
}

type PositionStatusResponse struct {
	Status    string               `json:"status"`
	Positions []PositionStatusItem `json:"positions"`
	Summary   string               `json:"summary"`
	RequestID string               `json:"request_id"`
}

type PositionStatusItem struct {
	Symbol           string    `json:"symbol"`
	Amount           float64   `json:"amount"`
	AmountRequested  float64   `json:"amount_requested"`
	MarginAmount     float64   `json:"margin_amount"`
	EntryPrice       float64   `json:"entry_price"`
	CurrentPrice     float64   `json:"current_price"`
	Side             string    `json:"side"`
	ProfitTotal      float64   `json:"profit_total"`
	ProfitRealized   float64   `json:"profit_realized"`
	ProfitUnrealized float64   `json:"profit_unrealized"`
	OpenedAt         string    `json:"opened_at"`
	DurationMin      int64     `json:"duration_min"`
	DurationSec      int64     `json:"duration_sec"`
	TakeProfits      []float64 `json:"take_profits"`
	StopLoss         float64   `json:"stop_loss"`
}

type TradeHistoryResponse struct {
	Status    string             `json:"status"`
	Trades    []TradeHistoryItem `json:"trades"`
	Summary   string             `json:"summary"`
	RequestID string             `json:"request_id"`
}

type TradeHistoryItem struct {
	Symbol       string                          `json:"symbol"`
	Side         string                          `json:"side"`
	Amount       float64                         `json:"amount"`
	MarginAmount float64                         `json:"margin_amount"`
	OpenedAt     time.Time                       `json:"opened_at"`
	ClosedAt     time.Time                       `json:"closed_at"`
	DurationSec  int64                           `json:"duration_sec"`
	Profit       float64                         `json:"profit"`
	StopLoss     float64                         `json:"stop_loss,omitempty"`
	TakeProfits  []float64                       `json:"take_profits,omitempty"`
	Timeline     []DashboardRiskPlanTimelineItem `json:"risk_plan_timeline,omitempty"`
}

type DecisionLatestResponse struct {
	Status         string                      `json:"status"`
	Symbol         string                      `json:"symbol"`
	SnapshotID     uint                        `json:"snapshot_id,omitempty"`
	Input          *decisionfmt.DecisionInput  `json:"input,omitempty"`
	Decision       *decisionfmt.DecisionReport `json:"decision,omitempty"`
	Report         string                      `json:"report"`
	ReportMarkdown string                      `json:"report_markdown"`
	ReportHTML     string                      `json:"report_html"`
	Summary        string                      `json:"summary"`
	RequestID      string                      `json:"request_id"`
}

type ConfigBundle struct {
	Symbol   config.SymbolConfig
	Strategy config.StrategyConfig
}

type scheduleToggleRequest struct {
	Enable *bool  `json:"enable"`
	Reason string `json:"reason,omitempty"`
}

type scheduleSymbolRequest struct {
	Symbol string `json:"symbol"`
	Enable bool   `json:"enable"`
}

type observeRequest struct {
	Symbol string `json:"symbol"`
}

type debugPlanInjectRequest struct {
	Symbol         string  `json:"symbol"`
	Direction      string  `json:"direction"`
	RiskPct        float64 `json:"risk_pct"`
	LeverageCap    float64 `json:"leverage_cap"`
	EntryOffsetPct float64 `json:"entry_offset_pct"`
	StopOffsetPct  float64 `json:"stop_offset_pct"`
	TP1OffsetPct   float64 `json:"tp1_offset_pct"`
	ExpiresSec     int64   `json:"expires_sec"`
}

type debugPlanInjectResponse struct {
	Status       string    `json:"status"`
	Symbol       string    `json:"symbol"`
	Direction    string    `json:"direction"`
	MarkPrice    float64   `json:"mark_price"`
	Entry        float64   `json:"entry"`
	StopLoss     float64   `json:"stop_loss"`
	TakeProfits  []float64 `json:"take_profits"`
	RiskPct      float64   `json:"risk_pct"`
	RiskDistance float64   `json:"risk_distance"`
	PositionID   string    `json:"position_id"`
	ExpiresAt    time.Time `json:"expires_at"`
	RequestID    string    `json:"request_id"`
}

type debugPlanStatusResponse struct {
	Status        string                   `json:"status"`
	Symbol        string                   `json:"symbol"`
	Plan          *execution.ExecutionPlan `json:"plan,omitempty"`
	ExternalID    string                   `json:"external_id,omitempty"`
	ClientOrderID string                   `json:"client_order_id,omitempty"`
	SubmittedAt   int64                    `json:"submitted_at,omitempty"`
	RequestID     string                   `json:"request_id"`
}

type debugPlanClearRequest struct {
	Symbol string `json:"symbol"`
}

type debugPlanClearResponse struct {
	Status    string `json:"status"`
	Symbol    string `json:"symbol"`
	Cleared   bool   `json:"cleared"`
	RequestID string `json:"request_id"`
}

type ObserveAgentPayload struct {
	Indicator any `json:"indicator"`
	Structure any `json:"structure"`
	Mechanics any `json:"mechanics"`
}

type ObserveProviderPayload struct {
	Indicator any `json:"indicator"`
	Structure any `json:"structure"`
	Mechanics any `json:"mechanics"`
}

type ObserveGatePayload struct {
	Tradeable      bool   `json:"tradeable"`
	DecisionAction string `json:"decision_action"`
	DecisionText   string `json:"decision_text"`
	Grade          int    `json:"grade"`
	Reason         string `json:"reason"`
	ReasonCode     string `json:"reason_code"`
	Direction      string `json:"direction"`
}

type ObserveInPositionPayload struct {
	Indicator provider.InPositionIndicatorOut `json:"indicator"`
	Structure provider.InPositionStructureOut `json:"structure"`
	Mechanics provider.InPositionMechanicsOut `json:"mechanics"`
	Summary   string                          `json:"summary"`
}

type ObserveResponse struct {
	Symbol         string                   `json:"symbol"`
	Status         string                   `json:"status"`
	Agent          ObserveAgentPayload       `json:"agent"`
	Provider       *ObserveProviderPayload   `json:"provider,omitempty"`
	Gate           ObserveGatePayload        `json:"gate"`
	InPosition     *ObserveInPositionPayload `json:"in_position,omitempty"`
	Report         string                   `json:"report,omitempty"`
	ReportMarkdown string                   `json:"report_markdown,omitempty"`
	ReportHTML     string                   `json:"report_html,omitempty"`
	Summary        string                   `json:"summary"`
	RequestID      string                   `json:"request_id"`
	SkippedExec    bool                     `json:"skipped_execution"`
	TraceID        string                   `json:"llm_trace_id,omitempty"`
}

type observeResponse = ObserveResponse

func (p ObserveAgentPayload) HasData() bool {
	return p.Indicator != nil || p.Structure != nil || p.Mechanics != nil
}

func (p ObserveAgentPayload) Map() map[string]any {
	if !p.HasData() {
		return nil
	}
	return map[string]any{
		"indicator": p.Indicator,
		"structure": p.Structure,
		"mechanics": p.Mechanics,
	}
}

func (p ObserveGatePayload) HasData() bool {
	return p.Tradeable || p.DecisionAction != "" || p.DecisionText != "" || p.Grade != 0 || p.Reason != "" || p.ReasonCode != "" || p.Direction != ""
}

func (p ObserveGatePayload) Map() map[string]any {
	if !p.HasData() {
		return nil
	}
	return map[string]any{
		"tradeable":       p.Tradeable,
		"decision_action": p.DecisionAction,
		"decision_text":   p.DecisionText,
		"grade":           p.Grade,
		"reason":          p.Reason,
		"reason_code":     p.ReasonCode,
		"direction":       p.Direction,
	}
}

type observeJobKey struct {
	Symbol     string   `json:"symbol"`
	Intervals  []string `json:"intervals"`
	KlineLimit int      `json:"kline_limit"`
}

type lastObserve struct {
	resp observeResponse
	at   time.Time
}
