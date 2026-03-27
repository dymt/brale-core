package decisionfmt

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"brale-core/internal/pkg/parseutil"
)

var reportTimeNow = time.Now

func parseStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func parseBoolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case uint64:
		return v != 0
	default:
		return false
	}
}

func parseFloatValue(value any) float64 {
	return parseutil.Float(value)
}

func parseFloatValueOK(value any) (float64, bool) {
	return parseutil.FloatOK(value)
}

func parseStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		items := make([]string, 0, len(v))
		for _, raw := range v {
			text := parseStringValue(raw)
			if text == "" {
				continue
			}
			items = append(items, text)
		}
		return items
	default:
		return nil
	}
}

func parseMapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if v, ok := value.(map[string]any); ok {
		return v
	}
	return nil
}

type directionConsensusMetrics struct {
	Score                 float64
	ScoreOK               bool
	ScoreThreshold        float64
	ScoreThresholdOK      bool
	Confidence            float64
	ConfidenceOK          bool
	ConfidenceThreshold   float64
	ConfidenceThresholdOK bool
	IndicatorScore        float64
	IndicatorScoreOK      bool
	StructureScore        float64
	StructureScoreOK      bool
	MechanicsScore        float64
	MechanicsScoreOK      bool
}

func parseDirectionConsensusMetrics(derived map[string]any) *directionConsensusMetrics {
	if len(derived) == 0 {
		return nil
	}
	raw, ok := derived["direction_consensus"]
	if !ok {
		return nil
	}
	consensus := parseMapValue(raw)
	if len(consensus) == 0 {
		return nil
	}
	out := &directionConsensusMetrics{}
	out.Score, out.ScoreOK = parseFloatValueOK(consensus["score"])
	out.ScoreThreshold, out.ScoreThresholdOK = parseFloatValueOK(consensus["score_threshold"])
	out.Confidence, out.ConfidenceOK = parseFloatValueOK(consensus["confidence"])
	out.ConfidenceThreshold, out.ConfidenceThresholdOK = parseFloatValueOK(consensus["confidence_threshold"])
	sources := parseMapValue(consensus["sources"])
	if len(sources) > 0 {
		indicator := parseMapValue(sources["indicator"])
		out.IndicatorScore, out.IndicatorScoreOK = parseFloatValueOK(indicator["score"])
		structure := parseMapValue(sources["structure"])
		out.StructureScore, out.StructureScoreOK = parseFloatValueOK(structure["score"])
		mechanics := parseMapValue(sources["mechanics"])
		out.MechanicsScore, out.MechanicsScoreOK = parseFloatValueOK(mechanics["score"])
	}
	return out
}

func formatReportTime() string {
	now := reportTimeNow()
	return now.Format("2006-01-02 15:04:05 MST -0700")
}

func formatCurrentPrice(report DecisionReport) string {
	price := extractCurrentPrice(report)
	if price <= 0 {
		return "—"
	}
	text := strconv.FormatFloat(price, 'f', 4, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "—"
	}
	return text
}

func extractCurrentPrice(report DecisionReport) float64 {
	if report.Gate.Derived == nil {
		return 0
	}
	val, ok := report.Gate.Derived["current_price"]
	if !ok || val == nil {
		return 0
	}
	return parseutil.Float(val)
}
