# 2026-04-16 MCP install defaults to SSE with Docker auto-start

## Summary

Change `bralectl mcp install` so it defaults to installing an SSE-based MCP client configuration instead of stdio. During install, if the target SSE endpoint is not reachable, automatically start the required Docker-backed MCP SSE service and wait for readiness before writing the client config. This auto-start behavior applies only during `mcp install` when the selected mode is `sse`.

## Goals

- Make `bralectl mcp install` default to SSE mode.
- Preserve explicit opt-in to stdio via `--mode stdio`.
- Automatically start the Docker-based MCP SSE service during install when SSE is unavailable.
- Keep `mcp serve` behavior unchanged.
- Produce clear error messages for Docker/Compose/startup failures.

## Non-goals

- No runtime auto-recovery outside `bralectl mcp install`.
- No changes to `bralectl mcp serve` defaults.
- No support for auto-start in stdio mode.
- No broad Docker orchestration abstraction beyond what this install flow needs.

## Current state

Today `bralectl mcp install` always writes a stdio client configuration:
- `internal/mcp/install.go` hardcodes `mcp serve --mode stdio` in install args.
- Claude Code install output writes `"type": "stdio"`.
- `mcp serve` already supports both `stdio` and `sse`, but install does not expose mode-aware output.

## User-facing behavior

### Default install

`bralectl mcp install`
- defaults to `--mode sse`
- checks whether the configured SSE endpoint is reachable
- if reachable: writes SSE client config immediately
- if unreachable: attempts to auto-start the Docker-backed SSE service, waits for readiness, then writes SSE client config

### Explicit stdio install

`bralectl mcp install --mode stdio`
- preserves current behavior
- does not attempt Docker auto-start
- writes stdio client config exactly as today

### Explicit SSE install

`bralectl mcp install --mode sse`
- same behavior as the default install path

## Proposed CLI/API changes

### CLI

Add `--mode` to `bralectl mcp install`:
- allowed values: `sse`, `stdio`
- default: `sse`

This mode is independent from `mcp serve --mode` and only affects install output and install-time environment preparation.

### Internal API

Extend `mcp.InstallOptions` with:
- `Mode string`

Normalize install mode with a helper similar to `normalizeMCPServeMode`.

## Install flow design

### 1. Resolve install options

During `prepareInstall`:
- normalize target
- normalize install mode
- resolve executable/config/system/index/audit paths as today
- resolve endpoint as today

### 2. Mode-specific prepared install data

Prepared install output should carry:
- normalized `mode`
- config payload details needed for stdio vs SSE client entries

For stdio:
- keep current command/args generation

For SSE:
- do not generate `mcp serve --mode stdio` args for the client config
- instead prepare endpoint-based client config

## Client config output

### Claude Code target

For `stdio`, keep current shape:

```json
{
  "type": "stdio",
  "command": "/abs/path/to/bralectl",
  "args": ["--endpoint", "http://127.0.0.1:9991", "mcp", "serve", "--mode", "stdio", ...],
  "env": {}
}
```

For `sse`, write URL-based config. The exact field names must match Claude Code’s SSE transport format used by the current harness. The implementation should use a dedicated builder function so target-specific JSON shape is isolated and testable.

Design requirement: the stored config must clearly represent a remote/SSE MCP server rather than a spawned stdio process.

### Other targets

Mode-aware config generation should also be target-aware:
- `stdio` targets keep their existing JSON/TOML output.
- `sse` targets should only be supported where the target format can express URL-based MCP servers.
- If a target does not support SSE install output yet, fail with a clear unsupported-mode-for-target error rather than silently downgrading to stdio.

Initial implementation priority is Claude Code because that is the primary path being changed.

## SSE readiness check

Before writing SSE config, probe the SSE endpoint.

### Probe behavior

- Use the configured install endpoint/derived SSE URL.
- Treat successful HTTP response from the SSE endpoint as ready.
- Apply a short timeout per request.
- Retry for a bounded readiness window only after auto-start is attempted.

Implementation should keep readiness probing in a small helper so it is easy to stub in tests.

## Docker auto-start behavior

Auto-start is only attempted when:
- install mode is `sse`, and
- SSE endpoint probe fails

### Startup strategy

Prefer repository-defined startup command(s) over embedding Compose topology all over the install flow.

Recommended implementation:
- Introduce a focused helper that runs the project’s canonical startup command for the MCP SSE service.
- The helper encapsulates the shell command details and returns structured errors.

Expected first implementation source of truth:
- reuse an existing project command if one already exists and is stable
- otherwise use a single well-defined docker compose invocation for the known MCP SSE service in this repository

The install code should call a helper like:
- `ensureSSEAvailable(...)`
  - probe
  - if down, `startSSEViaDocker(...)`
  - wait for readiness

### Preconditions checked before startup

- `docker` executable exists
- Docker daemon is reachable
- compose subcommand is available (or equivalent supported invocation)
- command is run from a context where the project compose file can be resolved

### Failure modes

Return explicit errors for:
- Docker not installed
- Docker daemon not running / unreachable
- compose unavailable
- compose service not found / startup command failed
- SSE readiness timeout after startup

Error messages should include the attempted command and the next step the user can take.

## Repository integration

To avoid fragile path guessing, startup helper should operate relative to the current repository layout and known `docker-compose.yml` in the repo root.

The implementation should keep compose resolution centralized in one place, not duplicated across CLI code and install code.

## Architecture changes

### `cmd/bralectl/mcp.go`
- add install `--mode` flag
- pass mode into `mcp.InstallOptions`

### `internal/mcp/install.go`
- add install-mode normalization
- extend `InstallOptions` and `preparedInstall`
- split config generation into mode-aware builders
- add SSE ensure/start orchestration entry point used only by install

### New helper(s)
Likely new internal helpers in `internal/mcp/` or a small adjacent package:
- readiness probe helper
- Docker startup helper

These helpers should be dependency-injectable enough for unit testing without requiring real Docker.

## Testing strategy

### Unit tests

Cover:
- default install mode resolves to `sse`
- explicit `stdio` mode preserves existing config output
- explicit `sse` mode generates SSE config for Claude Code
- unsupported target/mode combinations fail clearly
- SSE reachable path skips Docker startup
- SSE unreachable path triggers startup helper
- startup helper failure returns clear wrapped error
- readiness timeout returns clear error

### Command tests

Update CLI tests to verify:
- `mcp install` default flags imply SSE mode
- `--mode stdio` and `--mode sse` are accepted

### Scope of test doubles

Do not shell out to real Docker in unit tests. Use injected function variables/interfaces for:
- endpoint probing
- command execution
- readiness waiting

## Rollout and compatibility

- Existing users who explicitly install stdio can continue doing so.
- Existing `mcp serve` workflows remain intact.
- The breaking default is intentional: plain `mcp install` changes from stdio to SSE.
- Errors should guide users to `--mode stdio` when they want a no-Docker local process path.

## Open decisions resolved by this design

- Default install mode: `sse`
- Docker auto-start scope: install-time only
- stdio compatibility: retained behind explicit `--mode stdio`
- runtime fallback: not added

## Implementation notes

Keep the install flow narrow:
- mode selection
- if SSE, ensure availability
- write target config

Do not refactor `mcp serve` or broader MCP server internals unless required by this feature.
