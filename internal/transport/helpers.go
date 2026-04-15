package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"brale-core/internal/pkg/logging"

	"go.uber.org/zap"
)

func StartHTTPServer(ctx context.Context, addr string, handler http.Handler, logger *zap.Logger) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", addr, err)
	}
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		if serveErr := httpSrv.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Warn("http server failed", zap.Error(serveErr))
		}
	}()
	ShutdownServerOnContext(ctx, httpSrv, 5*time.Second)
	return httpSrv, nil
}

func ShutdownServerOnContext(ctx context.Context, server *http.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}

// WriteBody writes data to the HTTP response and logs write failures at debug level.
func WriteBody(w http.ResponseWriter, data []byte) {
	if _, err := w.Write(data); err != nil {
		logging.L().Debug("http write failed", zap.Error(err))
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	WriteBody(w, data)
}

func BuildRuntimeBaseURL(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	host, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		return "http://" + strings.TrimRight(trimmed, "/")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}
