package botruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"brale-core/internal/pkg/httpclient"
)

type Client struct {
	baseURL string
	http    *http.Client
}

type errorResponse struct {
	Code      int            `json:"code"`
	Msg       string         `json:"msg"`
	RequestID string         `json:"request_id"`
	Details   map[string]any `json:"details"`
}

type scheduleToggleRequest struct {
	Enable *bool `json:"enable"`
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return nil, errors.New("runtime base url is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{baseURL: trimmed, http: httpClient}, nil
}

func (c *Client) Do(ctx context.Context, method, path string, payload any, out any) error {
	req, err := httpclient.NewJSONRequest(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil {
			if strings.TrimSpace(apiErr.Msg) != "" {
				return errors.New(strings.TrimSpace(apiErr.Msg))
			}
			if apiErr.Code != 0 {
				return fmt.Errorf("runtime api error code=%d", apiErr.Code)
			}
		}
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) FetchMonitorStatus(ctx context.Context) (MonitorStatusResponse, error) {
	var out MonitorStatusResponse
	err := c.Do(ctx, http.MethodGet, "/api/runtime/monitor/status", nil, &out)
	return out, err
}

func (c *Client) FetchPositionStatus(ctx context.Context) (PositionStatusResponse, error) {
	var out PositionStatusResponse
	err := c.Do(ctx, http.MethodGet, "/api/runtime/position/status", nil, &out)
	return out, err
}

func (c *Client) FetchTradeHistory(ctx context.Context) (TradeHistoryResponse, error) {
	var out TradeHistoryResponse
	err := c.Do(ctx, http.MethodGet, "/api/runtime/position/history", nil, &out)
	return out, err
}

func (c *Client) FetchDecisionLatest(ctx context.Context, symbol string) (DecisionLatestResponse, error) {
	var out DecisionLatestResponse
	path := "/api/runtime/decision/latest?symbol=" + url.QueryEscape(symbol)
	err := c.Do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) FetchObserveReport(ctx context.Context, symbol string) (ObserveResponse, error) {
	var out ObserveResponse
	path := "/api/observe/report?symbol=" + url.QueryEscape(symbol)
	err := c.Do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) FetchScheduleStatus(ctx context.Context) (ScheduleResponse, error) {
	var out ScheduleResponse
	err := c.Do(ctx, http.MethodGet, "/api/runtime/schedule/status", nil, &out)
	return out, err
}

func (c *Client) FetchDashboardOverview(ctx context.Context, symbol string) (DashboardOverviewResponse, error) {
	var out DashboardOverviewResponse
	path := "/api/runtime/dashboard/overview"
	if symbol = strings.TrimSpace(symbol); symbol != "" {
		path += "?symbol=" + url.QueryEscape(symbol)
	}
	err := c.Do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) FetchDashboardAccountSummary(ctx context.Context) (DashboardAccountSummaryResponse, error) {
	var out DashboardAccountSummaryResponse
	err := c.Do(ctx, http.MethodGet, "/api/runtime/dashboard/account_summary", nil, &out)
	return out, err
}

func (c *Client) FetchDashboardKline(ctx context.Context, symbol, interval string, limit int) (DashboardKlineResponse, error) {
	var out DashboardKlineResponse
	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("interval", interval)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	err := c.Do(ctx, http.MethodGet, "/api/runtime/dashboard/kline?"+query.Encode(), nil, &out)
	return out, err
}

func (c *Client) FetchDashboardDecisionFlow(ctx context.Context, symbol string, snapshotID uint) (DashboardDecisionFlowResponse, error) {
	var out DashboardDecisionFlowResponse
	query := url.Values{}
	query.Set("symbol", symbol)
	if snapshotID > 0 {
		query.Set("snapshot_id", fmt.Sprintf("%d", snapshotID))
	}
	err := c.Do(ctx, http.MethodGet, "/api/runtime/dashboard/decision_flow?"+query.Encode(), nil, &out)
	return out, err
}

func (c *Client) FetchDashboardDecisionHistory(ctx context.Context, symbol string, limit int, snapshotID uint) (DashboardDecisionHistoryResponse, error) {
	var out DashboardDecisionHistoryResponse
	query := url.Values{}
	query.Set("symbol", symbol)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if snapshotID > 0 {
		query.Set("snapshot_id", fmt.Sprintf("%d", snapshotID))
	}
	err := c.Do(ctx, http.MethodGet, "/api/runtime/dashboard/decision_history?"+query.Encode(), nil, &out)
	return out, err
}

func (c *Client) PostScheduleToggle(ctx context.Context, enable bool) (ScheduleResponse, error) {
	var out ScheduleResponse
	req := scheduleToggleRequest{Enable: &enable}
	path := "/api/runtime/schedule/disable"
	if enable {
		path = "/api/runtime/schedule/enable"
	}
	err := c.Do(ctx, http.MethodPost, path, req, &out)
	return out, err
}

func (c *Client) RunObserve(ctx context.Context, req ObserveRunRequest) (ObserveResponse, error) {
	var out ObserveResponse
	err := c.Do(ctx, http.MethodPost, "/api/observe/run", req, &out)
	return out, err
}
