# Refactor Follow-Ups

## Remaining Structural Work

1. Finish `internal/readmodel` split beyond dashboard.
   - Add `internal/readmodel/decisionflow/` for decision detail and report assembly.
   - Add `internal/readmodel/portfolio/` for overview, account, and trade history projections.
   - Reduce `internal/transport/runtimeapi/*` to request validation plus JSON response mapping only.

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

1. Re-run full package validation after the remaining read-model and transport follow-ups are finished.
   - `go test ./...`
   - `go test -race ./...`

2. Add more characterization tests for refactored boundaries.
   - `internal/runtime/webhook_sync_service.go`
   - `internal/transport/runtimeapi/usecase_dashboard_flow.go`
