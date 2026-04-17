package suites

import (
	"context"
	"fmt"
	"strings"
	"time"

	"brale-core/internal/e2e"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	e2e.Register("mcp", func() e2e.Suite { return &MCPSuite{} })
}

// MCPSuite tests the MCP Streamable HTTP server by connecting as an MCP client
// and invoking deterministic, read-only tools.
type MCPSuite struct{}

func (s *MCPSuite) Name() string { return "mcp" }

func (s *MCPSuite) Run(ctx *e2e.Context) e2e.SuiteResult {
	start := time.Now()
	result := e2e.SuiteResult{
		Name:      s.Name(),
		StartedAt: start,
	}

	endpoint := strings.TrimSpace(ctx.Config.MCPEndpoint)
	if endpoint == "" {
		result.Status = e2e.StatusSkip
		result.Error = "mcp-endpoint not configured"
		result.Duration = time.Since(start)
		return result
	}

	allPassed := true

	// Connect to MCP Streamable HTTP server
	client := sdkmcp.NewClient(
		&sdkmcp.Implementation{Name: "brale-e2e", Version: "v1.0.0"},
		nil,
	)
	transport := &sdkmcp.StreamableClientTransport{Endpoint: endpoint}
	connCtx, connCancel := context.WithTimeout(ctx.Ctx, 15*time.Second)
	defer connCancel()
	session, err := client.Connect(connCtx, transport, nil)
	if err != nil {
		result.Checks = append(result.Checks, e2e.CheckResult{
			Name:    "mcp_connect",
			Passed:  false,
			Message: fmt.Sprintf("connect failed: %v", err),
		})
		result.Status = e2e.StatusFail
		result.Error = "MCP HTTP connection failed"
		result.Duration = time.Since(start)
		return result
	}
	defer session.Close()
	result.Checks = append(result.Checks, e2e.CheckResult{
		Name: "mcp_connect", Passed: true, Message: "connected",
	})

	// 1. List tools
	toolsCheck := checkListTools(ctx.Ctx, session)
	result.Checks = append(result.Checks, toolsCheck)
	if !toolsCheck.Passed {
		allPassed = false
	}

	// 2. List resources
	resourcesCheck := checkListResources(ctx.Ctx, session)
	result.Checks = append(result.Checks, resourcesCheck)
	if !resourcesCheck.Passed {
		allPassed = false
	}

	// 3. Call get_config (deterministic, no external deps)
	configCheck := checkCallGetConfig(ctx.Ctx, session)
	result.Checks = append(result.Checks, configCheck)
	if !configCheck.Passed {
		allPassed = false
	}

	// 4. Call get_positions (deterministic, returns list)
	posCheck := checkCallGetPositions(ctx.Ctx, session)
	result.Checks = append(result.Checks, posCheck)
	if !posCheck.Passed {
		allPassed = false
	}

	if allPassed {
		result.Status = e2e.StatusPass
	} else {
		result.Status = e2e.StatusFail
		result.Error = "one or more MCP checks failed"
	}
	result.Duration = time.Since(start)
	return result
}

func checkListTools(ctx context.Context, session *sdkmcp.ClientSession) e2e.CheckResult {
	check := e2e.CheckResult{Name: "tools/list"}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tools, err := session.ListTools(callCtx, nil)
	if err != nil {
		check.Message = fmt.Sprintf("error: %v", err)
		return check
	}
	expectedTools := []string{
		"analyze_market", "get_latest_decision", "get_positions",
		"get_decision_history", "get_account_summary", "get_kline",
		"compute_indicators", "get_config",
	}
	found := make(map[string]bool)
	for _, t := range tools.Tools {
		found[t.Name] = true
	}
	var missing []string
	for _, name := range expectedTools {
		if !found[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		check.Message = fmt.Sprintf("missing tools: %s (got %d)", strings.Join(missing, ","), len(tools.Tools))
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("%d tools registered", len(tools.Tools))
	return check
}

func checkListResources(ctx context.Context, session *sdkmcp.ClientSession) e2e.CheckResult {
	check := e2e.CheckResult{Name: "resources/list"}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resources, err := session.ListResources(callCtx, nil)
	if err != nil {
		check.Message = fmt.Sprintf("error: %v", err)
		return check
	}
	if len(resources.Resources) == 0 {
		check.Message = "no resources returned"
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("%d resources", len(resources.Resources))
	return check
}

func checkCallGetConfig(ctx context.Context, session *sdkmcp.ClientSession) e2e.CheckResult {
	check := e2e.CheckResult{Name: "call get_config"}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := session.CallTool(callCtx, &sdkmcp.CallToolParams{
		Name: "get_config",
	})
	if err != nil {
		check.Message = fmt.Sprintf("error: %v", err)
		return check
	}
	if len(res.Content) == 0 {
		check.Message = "empty content"
		return check
	}
	text, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok || strings.TrimSpace(text.Text) == "" {
		check.Message = "no text content"
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("config returned (%d chars)", len(text.Text))
	return check
}

func checkCallGetPositions(ctx context.Context, session *sdkmcp.ClientSession) e2e.CheckResult {
	check := e2e.CheckResult{Name: "call get_positions"}
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	res, err := session.CallTool(callCtx, &sdkmcp.CallToolParams{
		Name: "get_positions",
	})
	if err != nil {
		check.Message = fmt.Sprintf("error: %v", err)
		return check
	}
	if len(res.Content) == 0 {
		check.Message = "empty content"
		return check
	}
	check.Passed = true
	check.Message = fmt.Sprintf("positions returned (%d content items)", len(res.Content))
	return check
}
