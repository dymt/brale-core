# Refactor Follow-Ups

## Remaining Structural Work

1. Finish `internal/readmodel` split beyond dashboard.
   - Add `internal/readmodel/decisionflow/` for decision detail and report assembly.
   - Add `internal/readmodel/portfolio/` for overview, account, and trade history projections.
   - Reduce `internal/transport/runtimeapi/*` to request validation plus JSON response mapping only.

2. Complete `internal/decision/ruleflow/node_gate.go` rule extraction.
   - Move remaining allow/result/reason selection semantics into explicit rule tables.
   - Keep evaluator as orchestration only.
   - Add focused tests once package test baseline is repaired.

3. Repair `internal/decision/ruleflow` package test baseline.
   - `plan_builder_test.go` currently references missing `evaluateFlatRuleflow` and `flatRuleflowOptions`.
   - Restore or replace those helpers so full `go test ./internal/decision/ruleflow/...` can pass again.

## Deferred Cleanup

1. Refactor `internal/decision/decisionfmt/formatter_translate.go`.
   - Convert repetitive branching into table-driven formatting with small escape hatches.

2. Refactor bot transports.
   - `internal/transport/feishubot/bot.go`
   - `internal/transport/telegrambot/bot.go`
   - Extract command registry and callback routing.

3. Refactor exchange transport readability hotspots.
   - `internal/market/binance/futures.go`
   - Split by endpoint or concern without changing wire behavior.

## Verification Debt

1. Re-run full package validation after the `ruleflow` test baseline is fixed.
   - `go test ./...`
   - `go test -race ./...`

2. Add more characterization tests for refactored boundaries.
   - `internal/runtime/webhook_sync_service.go`
   - `internal/transport/runtimeapi/usecase_dashboard_flow.go`
   - `internal/decision/ruleflow/node_gate.go`
