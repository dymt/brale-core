// 本文件主要内容：定义 Provider 输出结构体与枚举类型。
package provider

import (
	"brale-core/internal/decision/decisionutil"
	"brale-core/internal/llm"
)

type ConfidenceLevel string

type ThreatLevel string

const (
	ConfidenceLow  ConfidenceLevel = "low"
	ConfidenceHigh ConfidenceLevel = "high"
)

const (
	ThreatLevelNone     ThreatLevel = "none"
	ThreatLevelLow      ThreatLevel = "low"
	ThreatLevelMedium   ThreatLevel = "medium"
	ThreatLevelHigh     ThreatLevel = "high"
	ThreatLevelCritical ThreatLevel = "critical"
)

var (
	confidenceValues = []ConfidenceLevel{
		ConfidenceLow,
		ConfidenceHigh,
	}
	threatLevelValues = []ThreatLevel{
		ThreatLevelNone,
		ThreatLevelLow,
		ThreatLevelMedium,
		ThreatLevelHigh,
		ThreatLevelCritical,
	}
	confidenceSet  = decisionutil.BuildEnumSet(confidenceValues)
	threatLevelSet = decisionutil.BuildEnumSet(threatLevelValues)
)

func init() {
	llm.RegisterEnum[ConfidenceLevel](decisionutil.EnumStrings(confidenceValues)...)
	llm.RegisterEnum[ThreatLevel](decisionutil.EnumStrings(threatLevelValues)...)
}

func (c *ConfidenceLevel) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[ConfidenceLevel](data, confidenceSet, "confidence")
	if err != nil {
		return err
	}
	*c = value
	return nil
}

func (t *ThreatLevel) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[ThreatLevel](data, threatLevelSet, "threat_level")
	if err != nil {
		return err
	}
	*t = value
	return nil
}

type SemanticSignal struct {
	Value      bool            `json:"value"`
	Confidence ConfidenceLevel `json:"confidence"`
	Reason     string          `json:"reason"`
}

type IndicatorProviderOut struct {
	MomentumExpansion bool   `json:"momentum_expansion"`
	Alignment         bool   `json:"alignment"`
	MeanRevNoise      bool   `json:"mean_rev_noise"`
	SignalTag         string `json:"signal_tag"`
}

type StructureProviderOut struct {
	ClearStructure bool   `json:"clear_structure"`
	Integrity      bool   `json:"integrity"`
	Reason         string `json:"reason"`
	SignalTag      string `json:"signal_tag"`
}

type MechanicsProviderOut struct {
	LiquidationStress SemanticSignal `json:"liquidation_stress"`
	SignalTag         string         `json:"signal_tag"`
}

type InPositionIndicatorOut struct {
	MomentumSustaining bool   `json:"momentum_sustaining"`
	DivergenceDetected bool   `json:"divergence_detected"`
	Reason             string `json:"reason"`
	MonitorTag         string `json:"monitor_tag"`
}

type InPositionStructureOut struct {
	Integrity   bool        `json:"integrity"`
	ThreatLevel ThreatLevel `json:"threat_level"`
	Reason      string      `json:"reason"`
	MonitorTag  string      `json:"monitor_tag"`
}

type InPositionMechanicsOut struct {
	AdverseLiquidation bool   `json:"adverse_liquidation"`
	CrowdingReversal   bool   `json:"crowding_reversal"`
	Reason             string `json:"reason"`
	MonitorTag         string `json:"monitor_tag"`
}
