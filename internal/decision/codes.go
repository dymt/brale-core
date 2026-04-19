package decision

type LLMRiskReasonCode string

func (c LLMRiskReasonCode) String() string {
	return string(c)
}

const (
	LLMRiskReasonModeMissing      LLMRiskReasonCode = "LLM_RISK_INIT_MODE_MISSING"
	LLMRiskReasonModeMismatch     LLMRiskReasonCode = "LLM_RISK_INIT_MODE_MISMATCH"
	LLMRiskReasonTransportFailure LLMRiskReasonCode = "LLM_RISK_INIT_TRANSPORT_FAILURE"
	LLMRiskReasonSchemaFailure    LLMRiskReasonCode = "LLM_RISK_INIT_SCHEMA_FAILURE"
	LLMRiskReasonRatioFailure     LLMRiskReasonCode = "LLM_RISK_INIT_RATIO_FAILURE"
	LLMRiskReasonDirectionFailure LLMRiskReasonCode = "LLM_RISK_INIT_DIRECTION_FAILURE"
)

type FlowTerminalReason string

func (r FlowTerminalReason) String() string {
	return string(r)
}

const (
	FlowTerminalReasonGateBlocked FlowTerminalReason = "gate_blocked"
	FlowTerminalReasonPlanEmitted FlowTerminalReason = "plan_emitted"
)

const (
	llmRiskReasonModeMissing      = "LLM_RISK_INIT_MODE_MISSING"
	llmRiskReasonModeMismatch     = "LLM_RISK_INIT_MODE_MISMATCH"
	llmRiskReasonTransportFailure = "LLM_RISK_INIT_TRANSPORT_FAILURE"
	llmRiskReasonSchemaFailure    = "LLM_RISK_INIT_SCHEMA_FAILURE"
	llmRiskReasonRatioFailure     = "LLM_RISK_INIT_RATIO_FAILURE"
	llmRiskReasonDirectionFailure = "LLM_RISK_INIT_DIRECTION_FAILURE"
)
