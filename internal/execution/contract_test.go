package execution

import (
	"reflect"
	"testing"
	"time"
)

type contractField struct {
	name string
	typ  reflect.Type
}

func assertContractShape(t *testing.T, typ reflect.Type, fields []contractField) {
	t.Helper()
	if typ.NumField() != len(fields) {
		t.Fatalf("%s field count=%d want=%d", typ.Name(), typ.NumField(), len(fields))
	}
	for i, want := range fields {
		got := typ.Field(i)
		if got.Name != want.name {
			t.Fatalf("%s field[%d] name=%s want=%s", typ.Name(), i, got.Name, want.name)
		}
		if got.Type != want.typ {
			t.Fatalf("%s.%s type=%s want=%s", typ.Name(), got.Name, got.Type, want.typ)
		}
	}
}

func TestExecutionPlanContractShape(t *testing.T) {
	assertContractShape(t, reflect.TypeOf(ExecutionPlan{}), []contractField{
		{name: "Symbol", typ: reflect.TypeOf("")},
		{name: "Valid", typ: reflect.TypeOf(false)},
		{name: "InvalidReason", typ: reflect.TypeOf("")},
		{name: "Direction", typ: reflect.TypeOf("")},
		{name: "Entry", typ: reflect.TypeOf(float64(0))},
		{name: "StopLoss", typ: reflect.TypeOf(float64(0))},
		{name: "TakeProfits", typ: reflect.TypeOf([]float64(nil))},
		{name: "TakeProfitRatios", typ: reflect.TypeOf([]float64(nil))},
		{name: "RiskPct", typ: reflect.TypeOf(float64(0))},
		{name: "PositionSize", typ: reflect.TypeOf(float64(0))},
		{name: "Leverage", typ: reflect.TypeOf(float64(0))},
		{name: "RMultiple", typ: reflect.TypeOf(float64(0))},
		{name: "Template", typ: reflect.TypeOf("")},
		{name: "PlanSource", typ: reflect.TypeOf("")},
		{name: "StrategyID", typ: reflect.TypeOf("")},
		{name: "SystemConfigHash", typ: reflect.TypeOf("")},
		{name: "StrategyConfigHash", typ: reflect.TypeOf("")},
		{name: "PositionID", typ: reflect.TypeOf("")},
		{name: "LLMRiskTrace", typ: reflect.TypeOf((*LLMRiskTrace)(nil))},
		{name: "RiskAnnotations", typ: reflect.TypeOf(RiskAnnotations{})},
		{name: "CreatedAt", typ: reflect.TypeOf(time.Time{})},
		{name: "ExpiresAt", typ: reflect.TypeOf(time.Time{})},
	})
}

func TestRiskAnnotationsContractShape(t *testing.T) {
	assertContractShape(t, reflect.TypeOf(RiskAnnotations{}), []contractField{
		{name: "StopSource", typ: reflect.TypeOf("")},
		{name: "StopReason", typ: reflect.TypeOf("")},
		{name: "RiskDistance", typ: reflect.TypeOf(float64(0))},
		{name: "ATR", typ: reflect.TypeOf(float64(0))},
		{name: "BufferATR", typ: reflect.TypeOf(float64(0))},
		{name: "MaxInvestPct", typ: reflect.TypeOf(float64(0))},
		{name: "MaxInvestAmt", typ: reflect.TypeOf(float64(0))},
		{name: "MaxLeverage", typ: reflect.TypeOf(float64(0))},
		{name: "LiqPrice", typ: reflect.TypeOf(float64(0))},
		{name: "MMR", typ: reflect.TypeOf(float64(0))},
		{name: "Fee", typ: reflect.TypeOf(float64(0))},
	})
}
