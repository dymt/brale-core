package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestOpenAIClientCallUsesJSONObjectFormat(t *testing.T) {
	var responseFormat map[string]any
	client := &OpenAIClient{
		Endpoint: "https://llm.example.com",
		Model:    "m",
		APIKey:   "k",
		Timeout:  time.Second,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			responseFormat, _ = req["response_format"].(map[string]any)
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		}),
	}

	if _, err := client.Call(context.Background(), "sys", "user"); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if responseFormat["type"] != "json_object" {
		t.Fatalf("response_format.type=%v want json_object", responseFormat["type"])
	}
}

func TestOpenAIClientCallStructuredUsesJSONSchemaFormatWhenEnabled(t *testing.T) {
	var responseFormat map[string]any
	client := &OpenAIClient{
		Endpoint:         "https://llm.example.com",
		Model:            "m",
		APIKey:           "k",
		Timeout:          time.Second,
		StructuredOutput: true,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			responseFormat, _ = req["response_format"].(map[string]any)
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		}),
	}
	schema := &JSONSchema{
		Name: "sample",
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ok": map[string]any{"type": "boolean"},
			},
			"required":             []string{"ok"},
			"additionalProperties": false,
		},
	}

	if _, err := client.CallStructured(context.Background(), "sys", "user", schema); err != nil {
		t.Fatalf("CallStructured() error = %v", err)
	}
	if responseFormat["type"] != "json_schema" {
		t.Fatalf("response_format.type=%v want json_schema", responseFormat["type"])
	}
	jsonSchema, _ := responseFormat["json_schema"].(map[string]any)
	if jsonSchema["name"] != "sample" {
		t.Fatalf("json_schema.name=%v want sample", jsonSchema["name"])
	}
}

func TestOpenAIClientCallStructuredFallsBackWhenDisabled(t *testing.T) {
	var responseFormat map[string]any
	client := &OpenAIClient{
		Endpoint:         "https://llm.example.com",
		Model:            "m",
		APIKey:           "k",
		Timeout:          time.Second,
		StructuredOutput: false,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			responseFormat, _ = req["response_format"].(map[string]any)
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		}),
	}
	if _, err := client.CallStructured(context.Background(), "sys", "user", &JSONSchema{Name: "ignored"}); err != nil {
		t.Fatalf("CallStructured() error = %v", err)
	}
	if responseFormat["type"] != "json_object" {
		t.Fatalf("response_format.type=%v want json_object", responseFormat["type"])
	}
}

func TestOpenAIClientCallRetriesOn502(t *testing.T) {
	var attempts atomic.Int32
	client := &OpenAIClient{
		Endpoint: "https://llm.example.com",
		Model:    "m",
		APIKey:   "k",
		Timeout:  time.Second,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			n := attempts.Add(1)
			if n == 1 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Status:     "502 Bad Gateway",
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"error":"upstream unavailable"}`)),
				}, nil
			}
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		}),
	}

	out, err := client.Call(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if out != `{"ok":true}` {
		t.Fatalf("output=%q want %q", out, `{"ok":true}`)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts=%d want 2", got)
	}
}

func TestOpenAIClientCallOnceReturnsRetryAfterFor429WithoutHTTPRetry(t *testing.T) {
	var attempts atomic.Int32
	client := &OpenAIClient{
		Endpoint: "https://llm.example.com",
		Model:    "m",
		APIKey:   "k",
		Timeout:  time.Second,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			attempts.Add(1)
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Status:     "429 Too Many Requests",
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"Retry-After":  []string{"3"},
				},
				Body: io.NopCloser(strings.NewReader(`{"error":"rate limited"}`)),
			}, nil
		}),
	}

	out, retryAfter, err := client.callOnce(context.Background(), "https://llm.example.com/chat/completions", []byte(`{"ok":true}`), time.Second)
	if err == nil {
		t.Fatalf("callOnce() error = nil, want rate limit error")
	}
	if out != "" {
		t.Fatalf("output=%q want empty", out)
	}
	if retryAfter != 3*time.Second {
		t.Fatalf("retryAfter=%v want 3s", retryAfter)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("attempts=%d want 1", got)
	}
	if !strings.Contains(err.Error(), "429 Too Many Requests") {
		t.Fatalf("error=%q should mention 429", err.Error())
	}
}

func TestOpenAIClientCallOnceRetriesTransportErrors(t *testing.T) {
	var attempts atomic.Int32
	client := &OpenAIClient{
		Endpoint: "https://llm.example.com",
		Model:    "m",
		APIKey:   "k",
		Timeout:  time.Second,
		HTTPClient: newTestHTTPClient(func(r *http.Request) (*http.Response, error) {
			n := attempts.Add(1)
			if n == 1 {
				return nil, fmt.Errorf("temporary network failure")
			}
			return jsonResponse(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`), nil
		}),
	}

	out, retryAfter, err := client.callOnce(context.Background(), "https://llm.example.com/chat/completions", []byte(`{"ok":true}`), time.Second)
	if err != nil {
		t.Fatalf("callOnce() error = %v", err)
	}
	if retryAfter != 0 {
		t.Fatalf("retryAfter=%v want 0", retryAfter)
	}
	if out != `{"ok":true}` {
		t.Fatalf("output=%q want %q", out, `{"ok":true}`)
	}
	if got := attempts.Load(); got != 2 {
		t.Fatalf("attempts=%d want 2", got)
	}
}
