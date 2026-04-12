package telegrambot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"brale-core/internal/cardimage"
	"brale-core/internal/transport/botruntime"

	"go.uber.org/zap"
)

func TestExecuteObserveFlatSendsImageAfterAsyncReport(t *testing.T) {
	t.Parallel()

	srv := newTelegramObserveTestServer(func(state *telegramObserveTestState, w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/observe/run":
			writeObserveJSON(w, botruntime.ObserveResponse{
				Symbol:      "ETHUSDT",
				Status:      "running",
				Summary:     "观察任务已提交",
				RequestID:   "job-1",
				SkippedExec: true,
			})
		case r.URL.Path == "/api/observe/report":
			state.mu.Lock()
			state.reportCalls++
			call := state.reportCalls
			state.mu.Unlock()
			if call < 2 {
				http.NotFound(w, r)
				return
			}
			writeObserveJSON(w, botruntime.ObserveResponse{
				Symbol:    "ETHUSDT",
				Status:    "ok",
				RequestID: "job-1",
				Summary:   "观察完成：方向不明",
				Agent: map[string]any{
					"indicator": map[string]any{"alignment": "false"},
				},
				Gate: map[string]any{
					"decision_action": "WAIT",
					"reason":          "DIRECTION_UNCLEAR",
				},
			})
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	bot := newTelegramObserveTestBot(t, srv, func(ctx context.Context, symbol string, snapshotID uint, gate map[string]any, agent map[string]any, title string) (*cardimage.ImageAsset, error) {
		return &cardimage.ImageAsset{Data: []byte("png"), Filename: "observe.png"}, nil
	})

	bot.executeObserveFlat(context.Background(), 42, "ETHUSDT")

	state := srv.State()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.documentCalls != 1 {
		t.Fatalf("expected 1 image send, got %d", state.documentCalls)
	}
	if len(state.messages) != 0 {
		t.Fatalf("expected no text fallback, got %#v", state.messages)
	}
}

func TestExecuteObserveFlatFallsBackToTextOnTimeout(t *testing.T) {
	t.Parallel()

	srv := newTelegramObserveTestServer(func(state *telegramObserveTestState, w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/observe/run":
			writeObserveJSON(w, botruntime.ObserveResponse{
				Symbol:      "ETHUSDT",
				Status:      "running",
				Summary:     "观察任务已提交",
				RequestID:   "job-timeout",
				SkippedExec: true,
			})
		case r.URL.Path == "/api/observe/report":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	})
	defer srv.Close()

	bot := newTelegramObserveTestBot(t, srv, func(ctx context.Context, symbol string, snapshotID uint, gate map[string]any, agent map[string]any, title string) (*cardimage.ImageAsset, error) {
		return &cardimage.ImageAsset{Data: []byte("png"), Filename: "observe.png"}, nil
	})
	bot.observeTimeout = 40 * time.Millisecond

	bot.executeObserveFlat(context.Background(), 42, "ETHUSDT")

	state := srv.State()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.documentCalls != 0 {
		t.Fatalf("expected no image send, got %d", state.documentCalls)
	}
	if len(state.messages) == 0 {
		t.Fatalf("expected text fallback, got none")
	}
	if got := state.messages[len(state.messages)-1]; !strings.Contains(got, "观察任务已提交") {
		t.Fatalf("unexpected fallback text: %q", got)
	}
}

func TestSendObserveResponseFallsBackWhenRenderFails(t *testing.T) {
	t.Parallel()

	srv := newTelegramObserveTestServer(func(state *telegramObserveTestState, w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer srv.Close()

	bot := newTelegramObserveTestBot(t, srv, func(ctx context.Context, symbol string, snapshotID uint, gate map[string]any, agent map[string]any, title string) (*cardimage.ImageAsset, error) {
		return nil, errors.New("render failed")
	})

	bot.sendObserveResponse(context.Background(), 42, ObserveResponse{
		Symbol: "ETHUSDT",
		Status: "ok",
		Agent:  map[string]any{"indicator": map[string]any{"alignment": "false"}},
		Gate:   map[string]any{"decision_action": "WAIT", "reason": "DIRECTION_UNCLEAR"},
	})

	state := srv.State()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.documentCalls != 0 {
		t.Fatalf("expected no image send, got %d", state.documentCalls)
	}
	if len(state.messages) != 1 {
		t.Fatalf("expected 1 text fallback, got %#v", state.messages)
	}
	if !strings.Contains(state.messages[0], "观察图片生成失败：render failed") {
		t.Fatalf("unexpected fallback text: %q", state.messages[0])
	}
}

type telegramObserveTestServer struct {
	*httptest.Server
	state *telegramObserveTestState
}

func (s *telegramObserveTestServer) State() *telegramObserveTestState {
	return s.state
}

type telegramObserveTestState struct {
	mu            sync.Mutex
	reportCalls   int
	documentCalls int
	messages      []string
}

func newTelegramObserveTestServer(runtimeHandler func(state *telegramObserveTestState, w http.ResponseWriter, r *http.Request)) *telegramObserveTestServer {
	state := &telegramObserveTestState{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bottoken/sendDocument":
			state.mu.Lock()
			state.documentCalls++
			state.mu.Unlock()
			writeTelegramBaseResponse(w)
		case "/bottoken/sendMessage":
			var payload sendMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			state.mu.Lock()
			state.messages = append(state.messages, payload.Text)
			state.mu.Unlock()
			writeTelegramMessageResponse(w)
		default:
			runtimeHandler(state, w, r)
		}
	})
	return &telegramObserveTestServer{Server: httptest.NewServer(handler), state: state}
}

func newTelegramObserveTestBot(t *testing.T, srv *telegramObserveTestServer, render func(context.Context, string, uint, map[string]any, map[string]any, string) (*cardimage.ImageAsset, error)) *Bot {
	t.Helper()

	rc, err := botruntime.NewClient(srv.URL, srv.Client())
	if err != nil {
		t.Fatalf("new runtime client: %v", err)
	}
	return &Bot{
		apiBase:              srv.URL,
		token:                "token",
		runtimeClient:        rc,
		client:               srv.Client(),
		logger:               zap.NewNop(),
		requestTimeout:       100 * time.Millisecond,
		observeTimeout:       120 * time.Millisecond,
		observePollInterval:  10 * time.Millisecond,
		renderRuntimePayload: render,
	}
}

func writeObserveJSON(w http.ResponseWriter, resp botruntime.ObserveResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeTelegramBaseResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func writeTelegramMessageResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
}
