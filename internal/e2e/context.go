package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"brale-core/internal/transport/botruntime"
)

// Context provides shared state and clients for all suites in a run.
type Context struct {
	Ctx        context.Context
	Cancel     context.CancelFunc
	Config     RunConfig
	Client     *botruntime.Client
	HTTPClient *http.Client

	// Baseline captures from preflight
	BaselineSnapshotID uint
	BaselineRoundID    uint
	BaselineHistoryN   int
	BaselineMemoryN    int
}

// NewContext creates a new E2E context from the given run config.
func NewContext(cfg RunConfig) (*Context, error) {
	client, err := botruntime.NewClient(cfg.Endpoint, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("create runtime client: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	return &Context{
		Ctx:        ctx,
		Cancel:     cancel,
		Config:     cfg,
		Client:     client,
		HTTPClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// Log prints a timestamped message to stdout.
func (c *Context) Log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[e2e %s] %s\n", time.Now().Format("15:04:05"), msg)
}

// Healthz checks that the brale runtime API is reachable.
func (c *Context) Healthz() error {
	var resp map[string]any
	err := c.Client.Do(c.Ctx, http.MethodGet, "/healthz", nil, &resp)
	if err != nil {
		return fmt.Errorf("healthz: %w", err)
	}
	if status, _ := resp["status"].(string); status != "ok" {
		return fmt.Errorf("healthz status: %s", status)
	}
	return nil
}

// MarketStatus fetches the price stream status for a symbol.
func (c *Context) MarketStatus(symbol string) (map[string]any, error) {
	var resp map[string]any
	path := "/api/runtime/market/status?symbol=" + symbol
	err := c.Client.Do(c.Ctx, http.MethodGet, path, nil, &resp)
	return resp, err
}

// LLMRounds fetches the latest LLM rounds.
func (c *Context) LLMRounds(symbol string, limit int) (map[string]any, error) {
	var resp map[string]any
	path := fmt.Sprintf("/api/llm/rounds?symbol=%s&limit=%d", symbol, limit)
	err := c.Client.Do(c.Ctx, http.MethodGet, path, nil, &resp)
	return resp, err
}

// ForceClosePosition sends a forceexit request via Freqtrade API.
func (c *Context) ForceClosePosition(symbol string) error {
	if c.Config.FTEndpoint == "" {
		return fmt.Errorf("freqtrade endpoint not configured (use --ft-endpoint)")
	}

	// First, list open trades to find the trade_id
	listURL := c.Config.FTEndpoint + "/api/v1/status"
	req, err := http.NewRequestWithContext(c.Ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return fmt.Errorf("build list request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("list trades: %w", err)
	}
	defer resp.Body.Close()

	var trades []struct {
		TradeID int    `json:"trade_id"`
		Pair    string `json:"pair"`
		IsOpen  bool   `json:"is_open"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return fmt.Errorf("decode trades: %w", err)
	}

	var tradeID int
	for _, t := range trades {
		if t.IsOpen && t.Pair == symbol {
			tradeID = t.TradeID
			break
		}
	}
	if tradeID == 0 {
		return fmt.Errorf("no open trade for %s on freqtrade", symbol)
	}

	// Send forceexit
	payload := map[string]any{"tradeid": fmt.Sprintf("%d", tradeID)}
	body, _ := json.Marshal(payload)
	exitURL := c.Config.FTEndpoint + "/api/v1/forceexit"
	exitReq, err := http.NewRequestWithContext(c.Ctx, http.MethodPost, exitURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build forceexit request: %w", err)
	}
	exitReq.Header.Set("Content-Type", "application/json")
	exitResp, err := c.HTTPClient.Do(exitReq)
	if err != nil {
		return fmt.Errorf("forceexit: %w", err)
	}
	defer exitResp.Body.Close()

	if exitResp.StatusCode >= 400 {
		return fmt.Errorf("forceexit status: %d", exitResp.StatusCode)
	}
	return nil
}
