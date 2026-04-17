package position

import (
	"context"
	"strings"
	"testing"

	"brale-core/internal/execution"
)

func TestSubmitOpenFromPlanRequiresStopLoss(t *testing.T) {
	executor := &countingOpenExecutor{}
	svc := &PositionService{
		Store:     &stubStore{},
		Executor:  executor,
		PlanCache: NewPlanCache(),
	}

	plan := execution.ExecutionPlan{
		PositionID:   "ETHUSDT-1",
		Symbol:       "ETHUSDT",
		Direction:    "long",
		Entry:        2100,
		StopLoss:     0,
		PositionSize: 0.1,
		Leverage:     2,
	}

	_, err := svc.SubmitOpenFromPlan(context.Background(), plan, 2100)
	if err == nil || !strings.Contains(err.Error(), "stop_loss is required") {
		t.Fatalf("expected stop_loss validation error, got %v", err)
	}
	if executor.placeCalls != 0 {
		t.Fatalf("expected PlaceOrder not called, got %d", executor.placeCalls)
	}
}

func TestSubmitOpenFromPlanWithStopLossPlacesOrder(t *testing.T) {
	executor := &countingOpenExecutor{}
	svc := &PositionService{
		Store:     &stubStore{},
		Executor:  executor,
		PlanCache: NewPlanCache(),
	}

	plan := execution.ExecutionPlan{
		PositionID:   "ETHUSDT-2",
		Symbol:       "ETHUSDT",
		Direction:    "long",
		Entry:        2100,
		StopLoss:     2050,
		PositionSize: 0.1,
		Leverage:     2,
	}

	_, err := svc.SubmitOpenFromPlan(context.Background(), plan, 2100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executor.placeCalls != 1 {
		t.Fatalf("expected PlaceOrder called once, got %d", executor.placeCalls)
	}
}

func TestSubmitOpenFromPlanRejectsStopBeyondLiquidation(t *testing.T) {
	executor := &countingOpenExecutor{}
	svc := &PositionService{
		Store:     &stubStore{},
		Executor:  executor,
		PlanCache: NewPlanCache(),
	}

	plan := execution.ExecutionPlan{
		PositionID:   "ETHUSDT-3",
		Symbol:       "ETHUSDT",
		Direction:    "long",
		Entry:        2100,
		StopLoss:     1000,
		PositionSize: 0.1,
		Leverage:     2,
	}

	_, err := svc.SubmitOpenFromPlan(context.Background(), plan, 2100)
	if err == nil || !strings.Contains(err.Error(), "beyond liquidation") {
		t.Fatalf("expected liquidation validation error, got %v", err)
	}
	if executor.placeCalls != 0 {
		t.Fatalf("expected PlaceOrder not called, got %d", executor.placeCalls)
	}
}

func TestSubmitOpenFromPlanUsesOrderEntryForLiquidationGuard(t *testing.T) {
	executor := &countingOpenExecutor{}
	svc := &PositionService{
		Store:     &stubStore{},
		Executor:  executor,
		PlanCache: NewPlanCache(),
	}

	plan := execution.ExecutionPlan{
		PositionID:   "ETHUSDT-4",
		Symbol:       "ETHUSDT",
		Direction:    "long",
		Entry:        100,
		StopLoss:     60,
		PositionSize: 1,
		Leverage:     2,
	}

	_, err := svc.SubmitOpenFromPlan(context.Background(), plan, 150)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if executor.placeCalls != 1 {
		t.Fatalf("expected PlaceOrder called once, got %d", executor.placeCalls)
	}
}

type countingOpenExecutor struct {
	stubExecutor
	placeCalls int
}

func (e *countingOpenExecutor) PlaceOrder(ctx context.Context, req execution.PlaceOrderReq) (execution.PlaceOrderResp, error) {
	e.placeCalls++
	return execution.PlaceOrderResp{ExternalID: "stub", Status: "submitted"}, nil
}
