// Package e2e provides a suite-based E2E test runner for brale-core.
// It drives the runtime API and Freqtrade API from outside the process,
// validating the full trading lifecycle in an isolated environment.
package e2e

import (
	"time"
)

// SuiteResult holds the result of a single suite execution.
type SuiteResult struct {
	Name      string        `json:"name"`
	Status    SuiteStatus   `json:"status"`
	Duration  time.Duration `json:"duration_ms"`
	Error     string        `json:"error,omitempty"`
	Checks    []CheckResult `json:"checks,omitempty"`
	Evidence  []Evidence    `json:"evidence,omitempty"`
	StartedAt time.Time     `json:"started_at"`
}

// SuiteStatus enumerates possible suite outcomes.
type SuiteStatus string

const (
	StatusPass    SuiteStatus = "PASS"
	StatusFail    SuiteStatus = "FAIL"
	StatusSkip    SuiteStatus = "SKIP"
	StatusBlocked SuiteStatus = "BLOCKED"
)

// CheckResult holds a single assertion result.
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// Evidence stores a piece of diagnostic evidence.
type Evidence struct {
	Label string `json:"label"`
	Data  any    `json:"data"`
}

// RunConfig holds the complete configuration for an E2E test run.
type RunConfig struct {
	Profile         string
	Suites          []string
	Symbol          string
	Endpoint        string
	FTEndpoint      string
	PGPort          string
	MCPEndpoint     string
	Timeout         time.Duration
	ForceCloseAfter time.Duration
	ReportDir       string
	Bootstrap       string
	TelegramToken   string
	TelegramChatID  int64
}

// RunReport aggregates all suite results for a test run.
type RunReport struct {
	RunID     string        `json:"run_id"`
	Profile   string        `json:"profile"`
	Symbol    string        `json:"symbol"`
	StartedAt time.Time     `json:"started_at"`
	Duration  time.Duration `json:"duration_ms"`
	Suites    []SuiteResult `json:"suites"`
	Passed    bool          `json:"passed"`
}
