package botruntime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientDoRuntimeRequestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/monitor/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","symbols":[{"symbol":"BTCUSDT"}],"summary":"ok","request_id":"rid-1"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchMonitorStatus(context.Background())
	if err != nil {
		t.Fatalf("fetch monitor status: %v", err)
	}
	if resp.Status != "ok" || len(resp.Symbols) != 1 || resp.Symbols[0].Symbol != "BTCUSDT" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientDoRuntimeRequestErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/monitor/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":4001,"msg":"invalid symbol","request_id":"rid-2"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.FetchMonitorStatus(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientFetchScheduleStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/schedule/status" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","llm_scheduled":true,"mode":"scheduled","next_runs":[],"summary":"ready","request_id":"rid-3"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchScheduleStatus(context.Background())
	if err != nil {
		t.Fatalf("fetch schedule status: %v", err)
	}
	if !resp.LLMScheduled || resp.Mode != "scheduled" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientFetchDashboardOverview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/dashboard/overview" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("symbol"); got != "BTCUSDT" {
			t.Fatalf("symbol query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","symbol":"BTCUSDT","symbols":[{"symbol":"BTCUSDT","position":{"side":"long"},"pnl":{"total":1.25},"reconciliation":{"status":"ok"}}],"summary":"contract_frozen","request_id":"rid-4"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchDashboardOverview(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("fetch dashboard overview: %v", err)
	}
	if resp.Symbol != "BTCUSDT" || len(resp.Symbols) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientFetchDashboardAccountSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/dashboard/account_summary" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","balance":{"currency":"USDT","total":1000},"profit":{"all_profit":12.5},"summary":"contract_frozen","request_id":"rid-5"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchDashboardAccountSummary(context.Background())
	if err != nil {
		t.Fatalf("fetch dashboard account summary: %v", err)
	}
	if resp.Balance.Currency != "USDT" || resp.Profit.AllProfit != 12.5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientFetchDashboardDecisionHistory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/dashboard/decision_history" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("symbol"); got != "ETHUSDT" {
			t.Fatalf("symbol query = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","symbol":"ETHUSDT","limit":5,"items":[{"snapshot_id":7,"action":"ALLOW","reason":"ok","at":"2026-04-13T00:00:00Z"}],"summary":"contract_frozen","request_id":"rid-6"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchDashboardDecisionHistory(context.Background(), "ETHUSDT", 5, 0)
	if err != nil {
		t.Fatalf("fetch dashboard decision history: %v", err)
	}
	if resp.Symbol != "ETHUSDT" || len(resp.Items) != 1 || resp.Items[0].SnapshotID != 7 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestClientFetchDashboardKline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/runtime/dashboard/kline" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("symbol"); got != "SOLUSDT" {
			t.Fatalf("symbol query = %q", got)
		}
		if got := r.URL.Query().Get("interval"); got != "1h" {
			t.Fatalf("interval query = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("limit query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","symbol":"SOLUSDT","interval":"1h","limit":10,"candles":[{"open_time":1,"close_time":2,"open":1,"high":2,"low":0.5,"close":1.5,"volume":10}],"summary":"contract_frozen","request_id":"rid-7"}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.FetchDashboardKline(context.Background(), "SOLUSDT", "1h", 10)
	if err != nil {
		t.Fatalf("fetch dashboard kline: %v", err)
	}
	if resp.Symbol != "SOLUSDT" || len(resp.Candles) != 1 || resp.Candles[0].Close != 1.5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
