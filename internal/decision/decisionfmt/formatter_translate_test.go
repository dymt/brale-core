package decisionfmt

import (
	"strings"
	"testing"
)

func TestTranslateDecisionActionTableDriven(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "allow", want: "允许"},
		{in: " TIGHTEN ", want: "收紧风控"},
		{in: "", want: ""},
		{in: "custom", want: "custom"},
	}
	for _, tc := range tests {
		if got := translateDecisionAction(tc.in); got != tc.want {
			t.Fatalf("translateDecisionAction(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestTranslateGateReasonSpecialCases(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "PASS_STRONG", want: "强通过"},
		{in: "AGENT_ERROR:model timeout", want: "Agent 阶段异常：model timeout"},
		{in: "PROVIDER_ERROR:data stale", want: "Provider 阶段异常：data stale"},
		{in: "GATE_ERROR:consensus", want: "Gate 阶段异常：consensus"},
		{in: "weird_reason", want: "weird_reason(英文)"},
	}
	for _, tc := range tests {
		if got := translateGateReason(tc.in); got != tc.want {
			t.Fatalf("translateGateReason(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestTranslateSieveReasonCodeFallsBackToGateReason(t *testing.T) {
	if got := translateSieveReasonCode("CROWD_ALIGN_LOW"); got != "同向拥挤/低置信" {
		t.Fatalf("translateSieveReasonCode mapped=%q", got)
	}
	if got := translateSieveReasonCode("PASS_WEAK"); got != "弱通过" {
		t.Fatalf("translateSieveReasonCode fallback=%q", got)
	}
}

func TestTranslateTermFallbacks(t *testing.T) {
	if got := translateTerm("trend_surge"); got != "trend_surge(趋势加速)" {
		t.Fatalf("translateTerm mapped=%q", got)
	}
	if got := translateTerm("中文"); got != "中文" {
		t.Fatalf("translateTerm han=%q", got)
	}
	if got := translateTerm("unknown_token"); got != "unknown_token(英文)" {
		t.Fatalf("translateTerm unknown=%q", got)
	}
}

func TestTranslateLLMKeyAndProviderRole(t *testing.T) {
	if got := translateLLMKey("confidence"); got != "置信度" {
		t.Fatalf("translateLLMKey=%q", got)
	}
	if got := translateLLMKey("custom_key"); got != "custom_key" {
		t.Fatalf("translateLLMKey custom=%q", got)
	}
	if got := translateLLMKey("cross_tf_summary"); got != "跨周期汇总" {
		t.Fatalf("translateLLMKey cross_tf=%q", got)
	}
	if got := translateLLMKey("movement_score"); got != "方向分数" {
		t.Fatalf("translateLLMKey movement_score=%q", got)
	}
	if got := providerRoleLabel("mechanics"); got != "市场机制" {
		t.Fatalf("providerRoleLabel=%q", got)
	}
	if got := providerRoleLabel(" custom "); got != "custom" {
		t.Fatalf("providerRoleLabel custom=%q", got)
	}
}

func TestTranslateLLMFieldRefs(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "pure chinese no refs",
			in:   "这是一段中文描述",
			want: "这是一段中文描述",
		},
		{
			name: "field=value",
			in:   "lower_tf_agreement=false",
			want: "低周期一致性=否",
		},
		{
			name: "dotted path=value",
			in:   "cross_tf_summary.alignment=mixed",
			want: "跨周期汇总.指标一致性=信号混杂/分歧",
		},
		{
			name: "mixed text with field ref",
			in:   "高 higher_tf_agreement=false，低 lower_tf_agreement=false",
			want: "高 高周期一致性=否，低 低周期一致性=否",
		},
		{
			name: "indicator state values",
			in:   "ema_stack=bull, bb_zone=near_upper",
			want: "EMA排列=多头排列, 布林带区间=靠近上轨",
		},
		{
			name: "standalone field path",
			in:   "movement_score 偏低",
			want: "方向分数 偏低",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TranslateLLMFieldRefs(tc.in)
			if got != tc.want {
				t.Errorf("TranslateLLMFieldRefs(%q)\n  got  = %q\n  want = %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsLLMFreeTextField(t *testing.T) {
	if !isLLMFreeTextField("momentum_detail") {
		t.Fatal("expected momentum_detail to be free text field")
	}
	if !isLLMFreeTextField("conflict_detail") {
		t.Fatal("expected conflict_detail to be free text field")
	}
	if isLLMFreeTextField("tradeable") {
		t.Fatal("expected tradeable to NOT be free text field")
	}
}

func TestTranslateTermIndicatorStates(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "above", want: "above(上方)"},
		{in: "below", want: "below(下方)"},
		{in: "bull", want: "bull(多头排列)"},
		{in: "bear", want: "bear(空头排列)"},
		{in: "squeeze", want: "squeeze(挤压收窄)"},
		{in: "trending", want: "trending(趋势行情)"},
		{in: "choppy", want: "choppy(震荡行情)"},
		{in: "oversold", want: "oversold(超卖)"},
		{in: "overbought", want: "overbought(超买)"},
	}
	for _, tc := range tests {
		if got := translateTerm(tc.in); got != tc.want {
			t.Fatalf("translateTerm(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatEventList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "events= with event key",
			in:   "events=price_cross_ema_fast_down",
			want: "事件=价格下穿快线EMA",
		},
		{
			name: "events含 with event key",
			in:   "events含 aroon_strong_bearish",
			want: "事件含 阿隆指标强势看空",
		},
		{
			name: "events= with multiple keys",
			in:   "events=price_cross_ema_mid_down,aroon_strong_bullish",
			want: "事件=价格下穿中线EMA, 阿隆指标强势看多",
		},
		{
			name: "no events pattern",
			in:   "some other text",
			want: "some other text",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatEventList(tc.in)
			if got != tc.want {
				t.Errorf("FormatEventList(%q)\n  got  = %q\n  want = %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeDirtyValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "aaroon_strong_bullish", want: "aroon_strong_bullish"},
		{in: "aaroon_strong_bearish", want: "aroon_strong_bearish"},
		{in: "45_55", want: "45-55"},
		{in: "normal_value", want: "normal_value"},
		{in: "bull", want: "bull"},
	}
	for _, tc := range tests {
		if got := normalizeDirtyValue(tc.in); got != tc.want {
			t.Fatalf("normalizeDirtyValue(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

// TestHalfTranslatedStringsRejected verifies that known bad outputs never appear.
// These are the exact error forms reported by users.
func TestHalfTranslatedStringsRejected(t *testing.T) {
	badOutputs := []string{
		"price_cross_中线EMA_down",
		"price_cross_快线EMA_down",
		"aroon_strong_看空",
		"aroon_strong_看多",
	}

	// These inputs, when translated via TranslateLLMFieldRefs, must NOT produce
	// any of the bad outputs.
	inputs := []string{
		"events=price_cross_ema_mid_down",
		"events=price_cross_ema_fast_down",
		"events含 aroon_strong_bearish",
		"events含 aroon_strong_bullish",
		"price_cross_ema_mid_down",
		"price_cross_ema_fast_down",
		"aroon_strong_bearish",
		"aroon_strong_bullish",
	}
	for _, input := range inputs {
		got := TranslateLLMFieldRefs(input)
		for _, bad := range badOutputs {
			if strings.Contains(got, bad) {
				t.Errorf("TranslateLLMFieldRefs(%q) produced half-translated output containing %q: %q",
					input, bad, got)
			}
		}
	}
}

func TestTranslateLLMFieldRefsWithEvents(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "events=event_key",
			in:   "events=price_cross_ema_fast_down",
			want: "事件=价格下穿快线EMA",
		},
		{
			name: "events含 event_key",
			in:   "events含 aroon_strong_bearish",
			want: "事件含 阿隆指标强势看空",
		},
		{
			name: "dirty value aaroon",
			in:   "aaroon_strong_bullish",
			want: "阿隆指标强势看多",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TranslateLLMFieldRefs(tc.in)
			if got != tc.want {
				t.Errorf("TranslateLLMFieldRefs(%q)\n  got  = %q\n  want = %q", tc.in, got, tc.want)
			}
		})
	}
}
