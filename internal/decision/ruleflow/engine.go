package ruleflow

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rulego/rulego"
	ruletypes "github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/filter"
)

type Engine struct {
	mu    sync.RWMutex
	cache map[string]ruletypes.RuleEngine
	pool  *rulego.RuleGo
}

func NewEngine() *Engine {
	return &Engine{cache: make(map[string]ruletypes.RuleEngine), pool: rulego.NewRuleGo()}
}

func (e *Engine) EnsureRegistered() error {
	for _, component := range []ruletypes.Node{
		&DeriveTradeableNode{},
		&filter.JsSwitchNode{},
		&GateEntryNode{},
		&GateDecisionNode{},
		&HardGuardNode{},
		&MonitorFusionNode{},
		&PlanBuilderNode{},
		&FSMDecisionNode{},
	} {
		if err := registerComponent(component); err != nil {
			return err
		}
	}
	return nil
}

func registerComponent(component ruletypes.Node) error {
	if err := rulego.Registry.Register(component); err != nil && !isComponentAlreadyExistsError(err) {
		return err
	}
	return nil
}

func isComponentAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "component already exists")
}

func (e *Engine) Evaluate(ctx context.Context, ruleChainPath string, input Input) (Result, error) {
	if strings.TrimSpace(ruleChainPath) == "" {
		return Result{}, fmt.Errorf("rule_chain is required")
	}
	payload, err := buildInputPayload(ctx, input)
	if err != nil {
		return Result{}, err
	}

	engine, err := e.loadEngine(ruleChainPath)
	if err != nil {
		return Result{}, err
	}
	var outMsg ruletypes.RuleMsg
	var outErr error
	var completed bool
	engine.OnMsgAndWait(ruletypes.NewMsgWithJsonData(payload), ruletypes.WithContext(ctx), ruletypes.WithOnEnd(func(ruleCtx ruletypes.RuleContext, msg ruletypes.RuleMsg, err error, relationType string) {
		outMsg = msg
		outErr = err
		completed = true
	}))
	if outErr != nil {
		return Result{}, outErr
	}
	if !completed {
		return Result{}, fmt.Errorf("ruleflow output missing")
	}
	data, err := outMsg.GetJsonData()
	if err != nil {
		return Result{}, err
	}
	resultRoot, ok := data.(map[string]any)
	if !ok {
		return Result{}, fmt.Errorf("ruleflow output invalid")
	}
	return parseResult(resultRoot)
}
