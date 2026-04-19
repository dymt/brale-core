// 本文件主要内容：定义 Agent 输出摘要与枚举类型。
package agent

import (
	"brale-core/internal/decision/decisionutil"
	"brale-core/internal/llm"
)

type Expansion string

type Alignment string

type Noise string

type Regime string

type LastBreak string

type Quality string

type Pattern string

type LeverageState string

type Crowding string

type RiskLevel string

const (
	ExpansionExpanding   Expansion = "expanding"
	ExpansionContracting Expansion = "contracting"
	ExpansionStable      Expansion = "stable"
	ExpansionMixed       Expansion = "mixed"
	ExpansionUnknown     Expansion = "unknown"
)

const (
	AlignmentAligned   Alignment = "aligned"
	AlignmentMixed     Alignment = "mixed"
	AlignmentDivergent Alignment = "divergent"
	AlignmentUnknown   Alignment = "unknown"
)

const (
	NoiseLow     Noise = "low"
	NoiseMedium  Noise = "medium"
	NoiseHigh    Noise = "high"
	NoiseMixed   Noise = "mixed"
	NoiseUnknown Noise = "unknown"
)

const (
	RegimeTrendUp   Regime = "trend_up"
	RegimeTrendDown Regime = "trend_down"
	RegimeRange     Regime = "range"
	RegimeMixed     Regime = "mixed"
	RegimeUnclear   Regime = "unclear"
)

const (
	LastBreakBosUp     LastBreak = "bos_up"
	LastBreakBosDown   LastBreak = "bos_down"
	LastBreakChochUp   LastBreak = "choch_up"
	LastBreakChochDown LastBreak = "choch_down"
	LastBreakNone      LastBreak = "none"
	LastBreakUnknown   LastBreak = "unknown"
)

const (
	QualityClean   Quality = "clean"
	QualityMessy   Quality = "messy"
	QualityMixed   Quality = "mixed"
	QualityUnclear Quality = "unclear"
)

const (
	PatternDoubleTop        Pattern = "double_top"
	PatternDoubleBottom     Pattern = "double_bottom"
	PatternHeadShoulders    Pattern = "head_shoulders"
	PatternInvHeadShoulders Pattern = "inv_head_shoulders"
	PatternTriangleSym      Pattern = "triangle_sym"
	PatternTriangleAsc      Pattern = "triangle_asc"
	PatternTriangleDesc     Pattern = "triangle_desc"
	PatternWedgeRising      Pattern = "wedge_rising"
	PatternWedgeFalling     Pattern = "wedge_falling"
	PatternFlag             Pattern = "flag"
	PatternPennant          Pattern = "pennant"
	PatternChannelUp        Pattern = "channel_up"
	PatternChannelDown      Pattern = "channel_down"
	PatternNone             Pattern = "none"
	PatternUnknown          Pattern = "unknown"
)

const (
	LeverageStateIncreasing LeverageState = "increasing"
	LeverageStateStable     LeverageState = "stable"
	LeverageStateOverheated LeverageState = "overheated"
	LeverageStateUnknown    LeverageState = "unknown"
)

const (
	CrowdingLong     Crowding = "long_crowded"
	CrowdingShort    Crowding = "short_crowded"
	CrowdingBalanced Crowding = "balanced"
	CrowdingUnknown  Crowding = "unknown"
)

const (
	RiskLevelLow     RiskLevel = "low"
	RiskLevelMedium  RiskLevel = "medium"
	RiskLevelHigh    RiskLevel = "high"
	RiskLevelUnknown RiskLevel = "unknown"
)

var (
	expansionValues = []Expansion{
		ExpansionExpanding,
		ExpansionContracting,
		ExpansionStable,
		ExpansionMixed,
		ExpansionUnknown,
	}
	alignmentValues = []Alignment{
		AlignmentAligned,
		AlignmentMixed,
		AlignmentDivergent,
		AlignmentUnknown,
	}
	noiseValues = []Noise{
		NoiseLow,
		NoiseMedium,
		NoiseHigh,
		NoiseMixed,
		NoiseUnknown,
	}
	regimeValues = []Regime{
		RegimeTrendUp,
		RegimeTrendDown,
		RegimeRange,
		RegimeMixed,
		RegimeUnclear,
	}
	lastBreakValues = []LastBreak{
		LastBreakBosUp,
		LastBreakBosDown,
		LastBreakChochUp,
		LastBreakChochDown,
		LastBreakNone,
		LastBreakUnknown,
	}
	qualityValues = []Quality{
		QualityClean,
		QualityMessy,
		QualityMixed,
		QualityUnclear,
	}
	patternValues = []Pattern{
		PatternDoubleTop,
		PatternDoubleBottom,
		PatternHeadShoulders,
		PatternInvHeadShoulders,
		PatternTriangleSym,
		PatternTriangleAsc,
		PatternTriangleDesc,
		PatternWedgeRising,
		PatternWedgeFalling,
		PatternFlag,
		PatternPennant,
		PatternChannelUp,
		PatternChannelDown,
		PatternNone,
		PatternUnknown,
	}
	leverageStateValues = []LeverageState{
		LeverageStateIncreasing,
		LeverageStateStable,
		LeverageStateOverheated,
		LeverageStateUnknown,
	}
	crowdingValues = []Crowding{
		CrowdingLong,
		CrowdingShort,
		CrowdingBalanced,
		CrowdingUnknown,
	}
	riskLevelValues = []RiskLevel{
		RiskLevelLow,
		RiskLevelMedium,
		RiskLevelHigh,
		RiskLevelUnknown,
	}
	expansionSet     = decisionutil.BuildEnumSet(expansionValues)
	alignmentSet     = decisionutil.BuildEnumSet(alignmentValues)
	noiseSet         = decisionutil.BuildEnumSet(noiseValues)
	regimeSet        = decisionutil.BuildEnumSet(regimeValues)
	lastBreakSet     = decisionutil.BuildEnumSet(lastBreakValues)
	qualitySet       = decisionutil.BuildEnumSet(qualityValues)
	patternSet       = decisionutil.BuildEnumSet(patternValues)
	leverageStateSet = decisionutil.BuildEnumSet(leverageStateValues)
	crowdingSet      = decisionutil.BuildEnumSet(crowdingValues)
	riskLevelSet     = decisionutil.BuildEnumSet(riskLevelValues)
	lastBreakAliasSet = map[string]struct{}{
		"break_up":   {},
		"break_down": {},
	}
)

func init() {
	llm.RegisterEnum[Expansion](decisionutil.EnumStrings(expansionValues)...)
	llm.RegisterEnum[Alignment](decisionutil.EnumStrings(alignmentValues)...)
	llm.RegisterEnum[Noise](decisionutil.EnumStrings(noiseValues)...)
	llm.RegisterEnum[Regime](decisionutil.EnumStrings(regimeValues)...)
	llm.RegisterEnum[LastBreak](decisionutil.EnumStrings(lastBreakValues)...)
	llm.RegisterEnum[Quality](decisionutil.EnumStrings(qualityValues)...)
	llm.RegisterEnum[Pattern](decisionutil.EnumStrings(patternValues)...)
	llm.RegisterEnum[LeverageState](decisionutil.EnumStrings(leverageStateValues)...)
	llm.RegisterEnum[Crowding](decisionutil.EnumStrings(crowdingValues)...)
	llm.RegisterEnum[RiskLevel](decisionutil.EnumStrings(riskLevelValues)...)
}

func (e *Expansion) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Expansion](data, expansionSet, "expansion")
	if err != nil {
		return err
	}
	*e = value
	return nil
}

func (a *Alignment) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Alignment](data, alignmentSet, "alignment")
	if err != nil {
		return err
	}
	*a = value
	return nil
}

func (n *Noise) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Noise](data, noiseSet, "noise")
	if err != nil {
		return err
	}
	*n = value
	return nil
}

func (r *Regime) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Regime](data, regimeSet, "regime")
	if err != nil {
		return err
	}
	*r = value
	return nil
}

func (b *LastBreak) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[LastBreak](data, lastBreakSet, "last_break")
	if err != nil {
		aliasValue, aliasErr := decisionutil.ParseEnumJSON(data, lastBreakAliasSet, "last_break")
		if aliasErr != nil {
			return err
		}
		value = LastBreak(normalizeLastBreak(aliasValue))
	}
	*b = value
	return nil
}

func normalizeLastBreak(value string) string {
	switch value {
	case "break_up":
		return string(LastBreakBosUp)
	case "break_down":
		return string(LastBreakBosDown)
	default:
		return value
	}
}

func (q *Quality) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Quality](data, qualitySet, "quality")
	if err != nil {
		return err
	}
	*q = value
	return nil
}

func (p *Pattern) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Pattern](data, patternSet, "pattern")
	if err != nil {
		return err
	}
	*p = value
	return nil
}

func (s *LeverageState) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[LeverageState](data, leverageStateSet, "leverage_state")
	if err != nil {
		return err
	}
	*s = value
	return nil
}

func (c *Crowding) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[Crowding](data, crowdingSet, "crowding")
	if err != nil {
		return err
	}
	*c = value
	return nil
}

func (r *RiskLevel) UnmarshalJSON(data []byte) error {
	value, err := decisionutil.UnmarshalEnumJSON[RiskLevel](data, riskLevelSet, "risk_level")
	if err != nil {
		return err
	}
	*r = value
	return nil
}

type IndicatorSummary struct {
	Expansion          Expansion `json:"expansion"`
	Alignment          Alignment `json:"alignment"`
	Noise              Noise     `json:"noise"`
	MomentumDetail     string    `json:"momentum_detail"`
	ConflictDetail     string    `json:"conflict_detail"`
	MovementScore      float64   `json:"movement_score"`
	MovementConfidence float64   `json:"movement_confidence"`
}

type StructureSummary struct {
	Regime             Regime    `json:"regime"`
	LastBreak          LastBreak `json:"last_break"`
	Quality            Quality   `json:"quality"`
	Pattern            Pattern   `json:"pattern"`
	VolumeAction       string    `json:"volume_action"`
	CandleReaction     string    `json:"candle_reaction"`
	MovementScore      float64   `json:"movement_score"`
	MovementConfidence float64   `json:"movement_confidence"`
}

type MechanicsSummary struct {
	LeverageState       LeverageState `json:"leverage_state"`
	Crowding            Crowding      `json:"crowding"`
	RiskLevel           RiskLevel     `json:"risk_level"`
	OpenInterestContext string        `json:"open_interest_context"`
	AnomalyDetail       string        `json:"anomaly_detail"`
	MovementScore       float64       `json:"movement_score"`
	MovementConfidence  float64       `json:"movement_confidence"`
}
