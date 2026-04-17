package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	sseProbeTimeout    = 2 * time.Second
	sseReadinessWindow = 30 * time.Second
	sseProbeInterval   = 1 * time.Second
)

var (
	httpClientFactory = func(timeout time.Duration) *http.Client {
		return &http.Client{Timeout: timeout}
	}
	lookPathFunc                 = exec.LookPath
	probeSSEFunc                 = probeSSE
	waitForSSEFunc               = waitForSSE
	checkDockerPrerequisitesFunc = checkDockerPrerequisites
	startSSEViaDockerFunc        = startSSEViaDocker
	runCommandFunc               = func(ctx context.Context, dir string, name string, args ...string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			if len(output) == 0 {
				return fmt.Errorf("%s %v: %w", name, args, err)
			}
			return fmt.Errorf("%s %v: %w: %s", name, args, err, string(output))
		}
		return nil
	}
)

func ensureSSEAvailable(prepared preparedInstall) error {
	if prepared.mode != "sse" {
		return nil
	}
	if err := probeSSEFunc(prepared.sseURL, sseProbeTimeout); err == nil {
		return nil
	}
	if err := checkDockerPrerequisitesFunc(prepared.repoRoot); err != nil {
		return err
	}
	if err := startSSEViaDockerFunc(prepared.repoRoot); err != nil {
		return err
	}
	if err := waitForSSEFunc(prepared.sseURL, sseReadinessWindow); err != nil {
		return err
	}
	return nil
}

func probeSSE(endpoint string, timeout time.Duration) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build SSE probe request: %w", err)
	}
	resp, err := httpClientFactory(timeout).Do(req)
	if err != nil {
		return fmt.Errorf("probe SSE endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("probe SSE endpoint %s: unexpected status %d", endpoint, resp.StatusCode)
	}
	return nil
}

func waitForSSE(endpoint string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	var lastErr error
	for {
		if err := probeSSE(endpoint, sseProbeTimeout); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(sseProbeInterval)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for SSE endpoint")
	}
	return fmt.Errorf("SSE readiness timeout for %s: %w", endpoint, lastErr)
}

func checkDockerPrerequisites(repoRoot string) error {
	if _, err := lookPathFunc("docker"); err != nil {
		return fmt.Errorf("docker executable not found: %w", err)
	}
	if err := runCommand(context.Background(), repoRoot, "docker", "info"); err != nil {
		return fmt.Errorf("docker daemon is not reachable (attempted: docker info): %w", err)
	}
	if err := runCommand(context.Background(), repoRoot, "docker", "compose", "version"); err != nil {
		return fmt.Errorf("docker compose is unavailable (attempted: docker compose version): %w", err)
	}
	composePath := filepath.Join(repoRoot, "docker-compose.yml")
	info, err := os.Stat(composePath)
	if err != nil {
		return fmt.Errorf("compose file not found at %s: %w", composePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("compose path is a directory, expected file: %s", composePath)
	}
	return nil
}

func startSSEViaDocker(repoRoot string) error {
	if err := runCommand(context.Background(), repoRoot, "docker", "compose", "--profile", "mcp", "up", "-d", "--build", "mcp-sse"); err != nil {
		return fmt.Errorf("start docker-backed MCP SSE service (attempted: docker compose --profile mcp up -d --build mcp-sse): %w", err)
	}
	return nil
}
