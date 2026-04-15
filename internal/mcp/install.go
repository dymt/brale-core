package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	defaultInstallTarget = "claude-code"
	defaultServerName    = "brale-core"
	defaultEndpoint      = "http://127.0.0.1:9991"
)

type InstallOptions struct {
	Name       string
	Command    string
	ConfigPath string
	Target     string
	Endpoint   string
	SystemPath string
	IndexPath  string
	AuditPath  string
}

type InstallResult struct {
	ConfigPath string
	ServerName string
	Command    string
	Args       []string
}

func Install(opts InstallOptions) (InstallResult, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = defaultServerName
	}
	command := strings.TrimSpace(opts.Command)
	if command == "" {
		exe, err := os.Executable()
		if err != nil {
			return InstallResult{}, fmt.Errorf("resolve executable: %w", err)
		}
		command = exe
	}
	command, err := absoluteExecutablePath(command)
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve command path: %w", err)
	}
	configPath := strings.TrimSpace(opts.ConfigPath)
	if configPath == "" {
		configPath, err = defaultInstallConfigPath(strings.TrimSpace(opts.Target))
		if err != nil {
			return InstallResult{}, err
		}
	}
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve config path: %w", err)
	}
	systemPath, err := absoluteExistingFile(defaultIfEmpty(opts.SystemPath, "configs/system.toml"))
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve system path: %w", err)
	}
	indexPath, err := absoluteExistingFile(defaultIfEmpty(opts.IndexPath, "configs/symbols-index.toml"))
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve index path: %w", err)
	}
	auditPath := strings.TrimSpace(opts.AuditPath)
	if auditPath == "" {
		auditPath, err = DefaultAuditLogPath()
		if err != nil {
			return InstallResult{}, fmt.Errorf("resolve audit log path: %w", err)
		}
	}
	auditPath, err = filepath.Abs(auditPath)
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve audit log path: %w", err)
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	args := []string{
		"--endpoint", endpoint,
		"mcp", "serve",
		"--mode", "stdio",
		"--system", systemPath,
		"--index", indexPath,
		"--audit-log", auditPath,
	}
	doc, err := loadInstallDocument(configPath)
	if err != nil {
		return InstallResult{}, err
	}
	servers, err := ensureMap(doc, "mcpServers")
	if err != nil {
		return InstallResult{}, err
	}
	servers[name] = map[string]any{
		"command": command,
		"args":    args,
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("create install dir: %w", err)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return InstallResult{}, fmt.Errorf("marshal install config: %w", err)
	}
	if err := writeAtomic(configPath, append(raw, '\n')); err != nil {
		return InstallResult{}, fmt.Errorf("write install config: %w", err)
	}
	return InstallResult{
		ConfigPath: configPath,
		ServerName: name,
		Command:    command,
		Args:       args,
	}, nil
}

func DefaultAuditLogPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "brale-core", "mcp-audit.jsonl"), nil
}

func defaultInstallConfigPath(target string) (string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "" {
		target = defaultInstallTarget
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch target {
	case "claude-code":
		if runtime.GOOS == "windows" {
			appData := strings.TrimSpace(os.Getenv("APPDATA"))
			if appData != "" {
				return filepath.Join(appData, "Claude", "mcp_settings.json"), nil
			}
		}
		return filepath.Join(home, ".config", "claude", "mcp_settings.json"), nil
	case "claude-desktop":
		switch runtime.GOOS {
		case "darwin":
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
		case "windows":
			appData := strings.TrimSpace(os.Getenv("APPDATA"))
			if appData != "" {
				return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
			}
			return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json"), nil
		default:
			return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
		}
	default:
		return "", fmt.Errorf("unsupported install target %q", target)
	}
}

func loadInstallDocument(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read install config: %w", err)
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse install config: %w", err)
	}
	if doc == nil {
		doc = map[string]any{}
	}
	return doc, nil
}

func ensureMap(doc map[string]any, key string) (map[string]any, error) {
	if existing, ok := doc[key]; ok {
		if typed, ok := existing.(map[string]any); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("%s must be a JSON object", key)
	}
	typed := map[string]any{}
	doc[key] = typed
	return typed, nil
}

func absoluteExecutablePath(path string) (string, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("command path must point to a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("command path must be executable")
	}
	return filepath.Abs(path)
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func absoluteExistingFile(path string) (string, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("must be a file path")
	}
	return filepath.Abs(path)
}

func writeAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if _, err := tmp.Write(content); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}
	return nil
}
