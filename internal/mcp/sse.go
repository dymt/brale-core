package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewSSEHandler(opts Options) (http.Handler, error) {
	server, err := NewServer(opts)
	if err != nil {
		return nil, err
	}
	return sdkmcp.NewSSEHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, nil), nil
}

func ServeSSE(ctx context.Context, opts Options, addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("sse addr is required")
	}
	handler, err := NewSSEHandler(opts)
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
