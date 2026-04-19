// 本文件主要职责：封装 Provider 阶段的 LLM 调用与输出解析。
// 本文件主要内容：调用 provider 模型并汇总持仓判断。

package llmapp

import (
	"context"
	"encoding/json"
	"fmt"

	"brale-core/internal/decision"
	"brale-core/internal/decision/agent"
	"brale-core/internal/decision/provider"
	"brale-core/internal/llm"
	"brale-core/internal/pkg/logging"
	"brale-core/internal/pkg/parallel"
	"brale-core/internal/prompt/positionprompt"

	"go.uber.org/zap"
)

type LLMProviderService struct {
	Runner  *provider.Runner
	Prompts LLMPromptBuilder
	Cache   *LLMStageCache
	Tracker *LLMRunTracker
}

type providerStageSpec[T any] struct {
	symbol        string
	logKind       string
	errorStage    string
	cacheStage    string
	stage         llm.LLMStage
	promptVersion string
	system        string
	userInput     string
	out           *T
	prompt        *decision.LLMStagePrompt
	invoke        func(context.Context, string, string) (T, error)
}

func (s LLMProviderService) Judge(ctx context.Context, symbol string, ind agent.IndicatorSummary, st agent.StructureSummary, mech agent.MechanicsSummary, enabled decision.AgentEnabled, dataCtx decision.ProviderDataContext) (provider.IndicatorProviderOut, provider.StructureProviderOut, provider.MechanicsProviderOut, decision.ProviderPromptSet, error) {
	if s.Runner == nil {
		logging.FromContext(ctx).Named("decision").Error("provider judge failed", zap.String("stage", "init"), zap.Error(fmt.Errorf("runner is required")))
		return provider.IndicatorProviderOut{}, provider.StructureProviderOut{}, provider.MechanicsProviderOut{}, decision.ProviderPromptSet{}, wrapLLMStageError("provider", symbol, "init", fmt.Errorf("runner is required"))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return provider.IndicatorProviderOut{}, provider.StructureProviderOut{}, provider.MechanicsProviderOut{}, decision.ProviderPromptSet{}, ctxErr
	}
	prompts, err := s.Prompts.ProviderPrompts(ind, st, mech, enabled, dataCtx)
	if err != nil {
		logging.FromContext(ctx).Named("decision").Error("provider judge failed", zap.String("stage", "prompts"), zap.Error(err))
		return provider.IndicatorProviderOut{}, provider.StructureProviderOut{}, provider.MechanicsProviderOut{}, decision.ProviderPromptSet{}, wrapLLMStageError("provider", symbol, "prompts", err)
	}
	var indOut provider.IndicatorProviderOut
	var stOut provider.StructureProviderOut
	var mechOut provider.MechanicsProviderOut
	var indPrompt decision.LLMStagePrompt
	var stPrompt decision.LLMStagePrompt
	var mechPrompt decision.LLMStagePrompt
	promptSet := decision.ProviderPromptSet{}
	tasks := make([]func(context.Context) error, 0, 3)
	if enabled.Indicator {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.IndicatorProviderOut]{
			symbol:        symbol,
			logKind:       "provider",
			errorStage:    "indicator",
			cacheStage:    "provider_indicator",
			stage:         llm.LLMStageIndicator,
			promptVersion: s.Prompts.ProviderIndicatorVersion,
			system:        prompts.IndicatorSys,
			userInput:     prompts.IndicatorUser,
			out:           &indOut,
			prompt:        &indPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.IndicatorProviderOut, error) {
				return s.Runner.JudgeIndicator(callCtx, system, user)
			},
		}))
	}
	if enabled.Structure {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.StructureProviderOut]{
			symbol:        symbol,
			logKind:       "provider",
			errorStage:    "structure",
			cacheStage:    "provider_structure",
			stage:         llm.LLMStageStructure,
			promptVersion: s.Prompts.ProviderStructureVersion,
			system:        prompts.StructureSys,
			userInput:     prompts.StructureUser,
			out:           &stOut,
			prompt:        &stPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.StructureProviderOut, error) {
				return s.Runner.JudgeStructure(callCtx, system, user)
			},
		}))
	}
	if enabled.Mechanics {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.MechanicsProviderOut]{
			symbol:        symbol,
			logKind:       "provider",
			errorStage:    "mechanics",
			cacheStage:    "provider_mechanics",
			stage:         llm.LLMStageMechanics,
			promptVersion: s.Prompts.ProviderMechanicsVersion,
			system:        prompts.MechanicsSys,
			userInput:     prompts.MechanicsUser,
			out:           &mechOut,
			prompt:        &mechPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.MechanicsProviderOut, error) {
				return s.Runner.JudgeMechanics(callCtx, system, user)
			},
		}))
	}
	if err := parallel.RunFailFast(ctx, tasks...); err != nil {
		return provider.IndicatorProviderOut{}, provider.StructureProviderOut{}, provider.MechanicsProviderOut{}, decision.ProviderPromptSet{}, err
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return provider.IndicatorProviderOut{}, provider.StructureProviderOut{}, provider.MechanicsProviderOut{}, decision.ProviderPromptSet{}, ctxErr
	}
	if enabled.Indicator {
		promptSet.Indicator = indPrompt
	}
	if enabled.Structure {
		promptSet.Structure = stPrompt
	}
	if enabled.Mechanics {
		promptSet.Mechanics = mechPrompt
	}
	logProviderDecision(ctx, symbol, enabled, indOut, stOut, mechOut)
	return indOut, stOut, mechOut, promptSet, nil
}

func (s LLMProviderService) JudgeInPosition(ctx context.Context, symbol string, ind agent.IndicatorSummary, st agent.StructureSummary, mech agent.MechanicsSummary, summary positionprompt.Summary, enabled decision.AgentEnabled, dataCtx decision.ProviderDataContext) (provider.InPositionIndicatorOut, provider.InPositionStructureOut, provider.InPositionMechanicsOut, decision.ProviderPromptSet, error) {
	logger := logging.FromContext(ctx).Named("decision").With(zap.String("symbol", symbol))
	if s.Runner == nil {
		logger.Error("provider judge failed", zap.String("stage", "init"), zap.Error(fmt.Errorf("runner is required")))
		return provider.InPositionIndicatorOut{}, provider.InPositionStructureOut{}, provider.InPositionMechanicsOut{}, decision.ProviderPromptSet{}, wrapLLMStageError("provider", symbol, "init", fmt.Errorf("runner is required"))
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return provider.InPositionIndicatorOut{}, provider.InPositionStructureOut{}, provider.InPositionMechanicsOut{}, decision.ProviderPromptSet{}, ctxErr
	}
	prompts, err := s.Prompts.InPositionProviderPrompts(ind, st, mech, summary, enabled, dataCtx)
	if err != nil {
		logger.Error("provider judge failed", zap.String("stage", "prompts"), zap.Error(err))
		return provider.InPositionIndicatorOut{}, provider.InPositionStructureOut{}, provider.InPositionMechanicsOut{}, decision.ProviderPromptSet{}, wrapLLMStageError("provider", symbol, "prompts_in_position", err)
	}
	var indOut provider.InPositionIndicatorOut
	var stOut provider.InPositionStructureOut
	var mechOut provider.InPositionMechanicsOut
	var indPrompt decision.LLMStagePrompt
	var stPrompt decision.LLMStagePrompt
	var mechPrompt decision.LLMStagePrompt
	promptSet := decision.ProviderPromptSet{}
	tasks := make([]func(context.Context) error, 0, 3)
	if enabled.Indicator {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.InPositionIndicatorOut]{
			symbol:        symbol,
			logKind:       "provider_in_position",
			errorStage:    "indicator_in_position",
			cacheStage:    "provider_indicator_in_position",
			stage:         llm.LLMStageIndicator,
			promptVersion: s.Prompts.ProviderInPosIndicatorVer,
			system:        prompts.IndicatorSys,
			userInput:     prompts.IndicatorUser,
			out:           &indOut,
			prompt:        &indPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.InPositionIndicatorOut, error) {
				return s.Runner.JudgeIndicatorInPosition(callCtx, system, user)
			},
		}))
	}
	if enabled.Structure {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.InPositionStructureOut]{
			symbol:        symbol,
			logKind:       "provider_in_position",
			errorStage:    "structure_in_position",
			cacheStage:    "provider_structure_in_position",
			stage:         llm.LLMStageStructure,
			promptVersion: s.Prompts.ProviderInPosStructureVer,
			system:        prompts.StructureSys,
			userInput:     prompts.StructureUser,
			out:           &stOut,
			prompt:        &stPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.InPositionStructureOut, error) {
				return s.Runner.JudgeStructureInPosition(callCtx, system, user)
			},
		}))
	}
	if enabled.Mechanics {
		tasks = append(tasks, providerStageTask(s, providerStageSpec[provider.InPositionMechanicsOut]{
			symbol:        symbol,
			logKind:       "provider_in_position",
			errorStage:    "mechanics_in_position",
			cacheStage:    "provider_mechanics_in_position",
			stage:         llm.LLMStageMechanics,
			promptVersion: s.Prompts.ProviderInPosMechanicsVer,
			system:        prompts.MechanicsSys,
			userInput:     prompts.MechanicsUser,
			out:           &mechOut,
			prompt:        &mechPrompt,
			invoke: func(callCtx context.Context, system string, user string) (provider.InPositionMechanicsOut, error) {
				return s.Runner.JudgeMechanicsInPosition(callCtx, system, user)
			},
		}))
	}
	if err := parallel.RunFailFast(ctx, tasks...); err != nil {
		return provider.InPositionIndicatorOut{}, provider.InPositionStructureOut{}, provider.InPositionMechanicsOut{}, decision.ProviderPromptSet{}, err
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return provider.InPositionIndicatorOut{}, provider.InPositionStructureOut{}, provider.InPositionMechanicsOut{}, decision.ProviderPromptSet{}, ctxErr
	}
	if enabled.Indicator {
		promptSet.Indicator = indPrompt
	}
	if enabled.Structure {
		promptSet.Structure = stPrompt
	}
	if enabled.Mechanics {
		promptSet.Mechanics = mechPrompt
	}
	logProviderInPositionDecision(ctx, symbol, enabled, indOut, stOut, mechOut)
	return indOut, stOut, mechOut, promptSet, nil
}

func (s LLMProviderService) providerUserPrompt(ctx context.Context, user string, symbol string, cacheStage string, stage llm.LLMStage) string {
	return appendLastOutput(user, s.Cache, symbol, cacheStage)
}

func runProviderWithLaneSession[T any](ctx context.Context, service LLMProviderService, symbol string, stage llm.LLMStage, user string, invoke func(context.Context, string, string) (T, error)) (T, string, error) {
	callCtx := llm.WithSessionSymbol(ctx, symbol)
	service.logLaneCall(callCtx, stage, "stateless", "", false, "")
	out, err := invoke(callCtx, "", user)
	return out, user, err
}

func (s LLMProviderService) logLaneCall(ctx context.Context, stage llm.LLMStage, mode string, sessionID string, reused bool, fallbackReason string) {
	logLLMLaneCall(ctx, "provider", stage, mode, sessionID, reused, fallbackReason)
}

func cacheProviderOutput(cache *LLMStageCache, symbol string, stage string, out any, input []byte) {
	cacheStageOutput(cache, symbol, stage, out, input)
}

func providerStageTask[T any](s LLMProviderService, spec providerStageSpec[T]) func(context.Context) error {
	return func(runCtx context.Context) error {
		runCtx = llm.WithSessionSymbol(runCtx, spec.symbol)
		system, user := applyMemoryPromptContext(runCtx, s.Prompts.UserFormat, spec.system, spec.userInput)
		cacheInput := promptCacheInput(system, user)
		user = s.providerUserPrompt(runCtx, user, spec.symbol, spec.cacheStage, spec.stage)
		*spec.prompt = decision.LLMStagePrompt{System: system, User: user, PromptVersion: spec.promptVersion}
		if out, ok := loadProviderCache[T](s.Cache, spec.symbol, spec.cacheStage, cacheInput); ok {
			*spec.out = out
			return nil
		}
		var collector stageCallCollector
		callCtx := withPromptCallContext(runCtx, spec.symbol, spec.logKind, spec.stage, spec.promptVersion, &collector)
		stageOut, finalUser, stageErr := runProviderWithLaneSession(callCtx, s, spec.symbol, spec.stage, user, func(callCtx context.Context, _ string, user string) (T, error) {
			return spec.invoke(callCtx, system, user)
		})
		spec.prompt.User = finalUser
		applyStageCallStats(spec.prompt, collector)
		if stageErr != nil {
			logging.FromContext(runCtx).Named("decision").Error("provider judge failed", zap.String("stage", spec.errorStage), zap.Error(stageErr))
			spec.prompt.Error = stageErr.Error()
			return wrapLLMStageError("provider", spec.symbol, spec.errorStage, stageErr)
		}
		*spec.out = stageOut
		if s.Tracker != nil {
			s.Tracker.MarkProvider()
		}
		cacheProviderOutput(s.Cache, spec.symbol, spec.cacheStage, stageOut, cacheInput)
		return nil
	}
}

func loadProviderCache[T any](cache *LLMStageCache, symbol string, stage string, input []byte) (T, bool) {
	if cache == nil {
		var zero T
		return zero, false
	}
	item, ok := cache.Load(symbol, stage, input)
	if !ok {
		var zero T
		return zero, false
	}
	var out T
	if err := json.Unmarshal(item.OutputJSON, &out); err != nil {
		var zero T
		return zero, false
	}
	return out, true
}

func logProviderDecision(ctx context.Context, symbol string, enabled decision.AgentEnabled, ind provider.IndicatorProviderOut, st provider.StructureProviderOut, mech provider.MechanicsProviderOut) {
	logger := logging.FromContext(ctx).Named("decision").With(zap.String("symbol", symbol))
	fields := make([]zap.Field, 0, 3)
	if enabled.Indicator {
		fields = append(fields, zap.String("指标", describeIndicator(ind)))
	} else {
		fields = append(fields, zap.String("指标", "禁用"))
	}
	if enabled.Structure {
		fields = append(fields, zap.String("结构", describeStructure(st)))
	} else {
		fields = append(fields, zap.String("结构", "禁用"))
	}
	if enabled.Mechanics {
		fields = append(fields, zap.String("机制", describeMechanics(mech)))
	} else {
		fields = append(fields, zap.String("机制", "禁用"))
	}
	logger.Info("provider LLM 决策", fields...)
}

func logProviderInPositionDecision(ctx context.Context, symbol string, enabled decision.AgentEnabled, ind provider.InPositionIndicatorOut, st provider.InPositionStructureOut, mech provider.InPositionMechanicsOut) {
	logger := logging.FromContext(ctx).Named("decision").With(zap.String("symbol", symbol))
	fields := make([]zap.Field, 0, 3)
	if enabled.Indicator {
		fields = append(fields, zap.String("指标", describeInPositionIndicator(ind)))
	} else {
		fields = append(fields, zap.String("指标", "禁用"))
	}
	if enabled.Structure {
		fields = append(fields, zap.String("结构", describeInPositionStructure(st)))
	} else {
		fields = append(fields, zap.String("结构", "禁用"))
	}
	if enabled.Mechanics {
		fields = append(fields, zap.String("机制", describeInPositionMechanics(mech)))
	} else {
		fields = append(fields, zap.String("机制", "禁用"))
	}
	logger.Info("provider LLM 决策(持仓)", fields...)
}

func describeIndicator(out provider.IndicatorProviderOut) string {
	return fmt.Sprintf(
		"momentum_expansion 动量扩张: %t; alignment 趋势一致: %t; mean_rev_noise 均值回归噪音: %t",
		out.MomentumExpansion,
		out.Alignment,
		out.MeanRevNoise,
	)
}

func describeStructure(out provider.StructureProviderOut) string {
	return fmt.Sprintf(
		"clear_structure 结构清晰: %t; integrity 叙事有效: %t; reason 理由: %s",
		out.ClearStructure,
		out.Integrity,
		out.Reason,
	)
}

func describeMechanics(out provider.MechanicsProviderOut) string {
	return fmt.Sprintf(
		"liquidation_stress 清算压力: %t; confidence 置信度: %s; reason 理由: %s",
		out.LiquidationStress.Value,
		string(out.LiquidationStress.Confidence),
		out.LiquidationStress.Reason,
	)
}

func describeInPositionIndicator(out provider.InPositionIndicatorOut) string {
	return fmt.Sprintf(
		"momentum_sustaining 动能维持: %t; divergence_detected 背离: %t; reason 理由: %s",
		out.MomentumSustaining,
		out.DivergenceDetected,
		out.Reason,
	)
}

func describeInPositionStructure(out provider.InPositionStructureOut) string {
	return fmt.Sprintf(
		"integrity 叙事有效: %t; threat_level 威胁等级: %s; reason 理由: %s",
		out.Integrity,
		string(out.ThreatLevel),
		out.Reason,
	)
}

func describeInPositionMechanics(out provider.InPositionMechanicsOut) string {
	return fmt.Sprintf(
		"adverse_liquidation 反向清算: %t; crowding_reversal 拥挤反转: %t; reason 理由: %s",
		out.AdverseLiquidation,
		out.CrowdingReversal,
		out.Reason,
	)
}
