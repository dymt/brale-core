package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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
