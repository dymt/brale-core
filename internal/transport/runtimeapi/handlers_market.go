package runtimeapi

import (
	"net/http"
	"strings"
	"time"

	"brale-core/internal/market"
)

func (s *Server) handleMarketStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !ensureMethod(ctx, w, r, http.MethodGet) {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("symbol")))
	if symbol == "" {
		writeError(ctx, w, http.StatusBadRequest, "missing_symbol", "symbol query parameter is required", nil)
		return
	}

	inspector, ok := s.PriceSource.(market.PriceStreamInspector)
	if !ok {
		writeJSON(w, map[string]any{
			"status":     "unsupported",
			"symbol":     symbol,
			"message":    "price source does not support stream inspection",
			"request_id": requestIDFromContext(ctx),
		})
		return
	}

	ss, found := inspector.StreamStatus(symbol)
	if !found {
		writeJSON(w, map[string]any{
			"status":     "not_found",
			"symbol":     symbol,
			"message":    "no stream data for this symbol",
			"request_id": requestIDFromContext(ctx),
		})
		return
	}

	resp := map[string]any{
		"status":          "ok",
		"symbol":          ss.Symbol,
		"source":          ss.Source,
		"ws_connected":    ss.Connected,
		"last_mark_price": ss.LastPrice,
		"age_ms":          ss.AgeMs,
		"fresh":           ss.Fresh,
		"request_id":      requestIDFromContext(ctx),
	}
	if !ss.LastPriceTS.IsZero() {
		resp["last_mark_ts"] = ss.LastPriceTS.Format(time.RFC3339Nano)
	}
	writeJSON(w, resp)
}
