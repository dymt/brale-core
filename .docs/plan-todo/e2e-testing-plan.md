# Brale-Core E2E Testing Plan

> **Last updated**: 2026-04-18
> **Status**: Executable — aligned with current codebase
> **Scope**: Automated E2E regression in Freqtrade dry-run mode

---

## 1. Overview

This plan describes how to run end-to-end tests against brale-core using the isolated
E2E Docker stack. All tests run in **Freqtrade dry-run mode** — they validate system
correctness (infrastructure, data flow, lifecycle), **not** profitability.

### Endpoints

| Service             | Endpoint                               |
|---------------------|----------------------------------------|
| Brale               | `http://127.0.0.1:19991`               |
| Brale health        | `http://127.0.0.1:19991/healthz`       |
| Freqtrade           | `http://127.0.0.1:18080`               |
| Freqtrade ping      | `http://127.0.0.1:18080/api/v1/ping`   |
| MCP (Streamable HTTP) | `http://127.0.0.1:18765/mcp`         |
| PostgreSQL          | `127.0.0.1:15432`                      |

> The E2E stack uses separate ports so it does not interfere with the main stack.

### CLI Entry Points

```bash
bralectl test list                           # List available test suites
bralectl test run --suites ctl               # Quick regression
bralectl test run --suites ctl,decision      # Multi-suite run
```

### Makefile Targets

```bash
make e2e-start                               # Start isolated E2E stack
make e2e-test                                # Quick regression (ctl suite)
make e2e-test E2E_SUITE=ctl,decision         # Run specific suites
make e2e-stop                                # Stop E2E stack
make e2e-reset                               # Clean all E2E data
```

---

## 2. Test Suites

| Suite                    | Description                                                     |
|--------------------------|-----------------------------------------------------------------|
| `ctl`                    | Basic connectivity: health, DB, Freqtrade ping, scheduler       |
| `decision`               | Observe + decide pipeline: triggers observation, verifies output |
| `reconcile`              | Position state sync between Brale and Freqtrade                 |
| `risk`                   | RiskMonitor, stop-loss checking, tighten logic                  |
| `mcp`                    | MCP Streamable HTTP: connect, tools/list, resources/list, tool calls |
| `open-fast`              | Fast open via debug plan inject                                 |
| `pricing`                | Mark price feed and price tick validation                       |
| `post-close`             | Post-close data integrity and episodic memory generation        |
| `lifecycle-force-close`  | Full lifecycle: inject plan → open → sync → force exit → close  |
| `lifecycle-natural`      | Full lifecycle driven by natural LLM decisions                  |

---

## 3. Phase A: Smoke — Infrastructure & Observability

> **Duration**: ~5 minutes
> **Dependency**: E2E stack is running (`make e2e-start`)

### 3.1 Docker Services Health

```bash
make status
# Expected: timescaledb healthy, freqtrade healthy, brale running
```

### 3.2 Database Connectivity

```bash
docker compose exec -T timescaledb psql -U brale -d brale \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
# Expected: table count > 10

docker compose exec -T timescaledb psql -U brale -d brale \
  -c "SELECT version, dirty FROM schema_migrations;"
# Expected: dirty = false
```

### 3.3 Brale Health

```bash
curl -sf http://127.0.0.1:19991/healthz
# Expected: 200 OK
```

### 3.4 Freqtrade API

```bash
curl -sf http://127.0.0.1:18080/api/v1/ping
# Expected: {"status":"pong"}
```

### 3.5 LLM Endpoints

```bash
bralectl llm probe
# Expected: all stages OK
```

### 3.6 Observation & Decision

```bash
bralectl observe run --symbol SOLUSDT --yes
bralectl observe report --symbol SOLUSDT
bralectl decision latest --symbol SOLUSDT
bralectl schedule status
```

### 3.7 MCP (Streamable HTTP)

```bash
curl -sf -X POST http://127.0.0.1:18765/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
# Expected: returns tool list with 9 registered tools
```

Or via CLI:

```bash
bralectl test run --suites mcp \
  --endpoint http://127.0.0.1:19991 \
  --mcp-endpoint http://127.0.0.1:18765
```

### Smoke Pass Criteria

- All containers running and healthy
- Database accessible, migration clean
- Brale `/healthz` returns 200
- Freqtrade `/ping` returns pong
- `bralectl llm probe` all OK
- Observe and decision commands return data
- MCP `tools/list` returns expected tools

---

## 4. Phase B: Lifecycle — Open → Sync → Close

> **Duration**: ~5 minutes
> **Dependency**: Phase A passes
> **Method**: Uses `/api/debug/plan/inject` to bypass natural market triggers

### 4.1 Inject a Debug Plan

```bash
curl -s -X POST http://127.0.0.1:19991/api/debug/plan/inject \
  -H "Content-Type: application/json" \
  -d '{
    "symbol":"SOLUSDT",
    "direction":"long",
    "risk_pct":0.01,
    "leverage_cap":2,
    "entry_offset_pct":0,
    "stop_offset_pct":0.01,
    "tp1_offset_pct":0.02,
    "expires_sec":300
  }'
```

### 4.2 Verify Plan Submission

Poll until `external_id` is non-empty (timeout: 120s):

```bash
curl -s "http://127.0.0.1:19991/api/debug/plan/status?symbol=SOLUSDT"
```

### 4.3 Confirm Position Open

```bash
bralectl position list
curl -s http://127.0.0.1:18080/api/v1/status
```

**Pass criteria:**
- `bralectl position list` shows the position
- Freqtrade `/status` shows the matching dry-run trade

### 4.4 Force Exit

Retrieve `trade_id` from Freqtrade status, then:

```bash
curl -s -X POST http://127.0.0.1:18080/api/v1/forceexit \
  -H "Content-Type: application/json" \
  -d '{"tradeid":"<trade_id>"}'
```

### 4.5 Verify Close

```bash
bralectl position history
curl -s http://127.0.0.1:18080/api/v1/status
```

**Pass criteria:**
- Freqtrade `/status` no longer includes the trade
- `bralectl position history` shows the closed record
- Brale logs contain open/close/reconcile entries

---

## 5. Phase C: Regression & Data Integrity

> **Duration**: ~3 minutes
> **Dependency**: Phase B passes (at least one complete open→close cycle)

### 5.1 Gate Replay

```bash
bralectl backtest rules --symbol SOLUSDT --from 2026-04-15 --to 2026-04-18
```

### 5.2 Indicator Diff

```bash
bralectl indicator diff --symbol SOLUSDT
```

### 5.3 Error Audit

```bash
docker compose logs brale 2>&1 | grep -Ei "panic|fatal" && exit 1 || echo "No panics or fatals"
```

### 5.4 Episodic Memory (if enabled)

```bash
bralectl memory episodic --symbol SOLUSDT
```

### Regression Pass Criteria

- Gate replay runs without error
- Indicator diff within acceptable tolerance (< 0.1%)
- No `panic` or `fatal` in logs
- Database state is consistent (no orphan open positions)

---

## 6. Running via Makefile

### Quick regression (recommended for CI)

```bash
make e2e-start
make e2e-test                                # runs ctl suite
make e2e-stop
```

### Full regression

```bash
make e2e-start
make e2e-test E2E_SUITE=ctl,decision,reconcile,risk,mcp
make e2e-stop
```

### Full lifecycle

```bash
make e2e-start
make e2e-test E2E_SUITE=lifecycle-force-close
make e2e-stop
```

### With Telegram notification

```bash
make e2e-start
make e2e-test E2E_SUITE=ctl,decision,reconcile E2E_TELEGRAM_NOTIFY=1
make e2e-stop
```

---

## 7. Running via bralectl

```bash
# List suites
bralectl test list

# Quick regression
bralectl test run --suites ctl

# Multi-suite with custom ports
bralectl test run --suites ctl,decision,mcp \
  --ft-endpoint http://127.0.0.1:18080 \
  --mcp-endpoint http://127.0.0.1:18765

# Full lifecycle with timeout
bralectl test run --suites lifecycle-force-close \
  --timeout 30m --force-close-after 5m

# With Telegram report
BRALE_E2E_TELEGRAM_TOKEN=123456:ABC... BRALE_E2E_TELEGRAM_CHAT_ID=-1001234567890 \
  bralectl test run --suites ctl,decision,reconcile
```

---

## 8. Key Metrics Reference

| Metric                          | Normal Range  | Alert Threshold |
|---------------------------------|---------------|-----------------|
| LLM single-call latency        | 3–15 s        | > 30 s          |
| LLM single-round tokens (input)| 10–13 K       | > 20 K          |
| Decision total time             | 10–45 s       | > 120 s         |
| Price tick interval             | 1–3 s         | > 30 s          |
| Reconcile interval              | 30 s          | > 5 min         |
| Container memory (brale)        | 100–300 MB    | > 1 GB          |
| DB size daily growth            | 1–10 MB       | > 100 MB        |
| Error logs per hour             | 0–5           | > 50            |

---

## 9. MCP E2E Suite Details

The `mcp` suite validates MCP Streamable HTTP transport (endpoint: `/mcp`).

| Check               | Description                                                     |
|---------------------|-----------------------------------------------------------------|
| `mcp_connect`       | Streamable HTTP connection established                          |
| `tools/list`        | Verifies 9 registered tools (analyze_market, get_latest_decision, get_positions, get_decision_history, get_account_summary, get_kline, compute_indicators, get_config, get_episodic_memory) |
| `resources/list`    | Verifies at least 1 registered resource                         |
| `call get_config`   | Calls get_config, validates non-empty TextContent               |
| `call get_positions`| Calls get_positions, validates returned content                 |

### Implementation

| File                            | Role                                           |
|---------------------------------|------------------------------------------------|
| `internal/e2e/suites/mcp.go`   | MCP suite using StreamableClientTransport       |
| `internal/e2e/types.go`        | RunConfig with `MCPEndpoint` field              |
| `cmd/bralectl/test_command.go`  | `--mcp-endpoint` flag                          |
| `Makefile`                      | e2e-test passes `--mcp-endpoint`; e2e-start launches mcp container |

If `--mcp-endpoint` is not specified, the MCP suite returns `SKIP` without affecting other suites.

---

## 10. Known Limitations

1. **Natural open/close timing** — depends on market conditions and LLM decisions; use `lifecycle-force-close` for deterministic testing
2. **Notification verification** — Telegram/Feishu receipt requires manual inspection
3. **Slippage and execution quality** — requires real trading data
4. **Multi-symbol stress testing** — requires configuring multiple symbols and extended runtime

For deterministic lifecycle testing, always prefer the debug inject path (`/api/debug/plan/inject`) over waiting for natural LLM triggers.
