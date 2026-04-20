// 本文件主要内容：维护 Binance Mark Price WebSocket 订阅与本地缓存。
package binance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"brale-core/internal/market"
	"brale-core/internal/pkg/logging"
	"brale-core/internal/pkg/parseutil"

	"github.com/adshao/go-binance/v2/futures"
	"go.uber.org/zap"
)

const (
	defaultMarkPriceMaxAge   = 30 * time.Second
	defaultMarkPriceSpikePct = 0.30
)

type MarkPriceStreamOptions struct {
	Symbols []string
	Rate    time.Duration
}

// MarkPriceStream keeps a local cache of mark prices from Binance futures websocket.
type MarkPriceStream struct {
	symbols []string
	rate    time.Duration

	mu     sync.RWMutex
	quotes map[string]market.PriceQuote

	ctrlMu    sync.Mutex
	stopCh    chan struct{}
	cancel    context.CancelFunc
	runID     atomic.Uint64
	running   atomic.Bool
	connected atomic.Bool
}

func NewMarkPriceStream(opts MarkPriceStreamOptions) (*MarkPriceStream, error) {
	symbols := normalizeSymbols(opts.Symbols)
	rate, err := normalizeMarkPriceRate(opts.Rate)
	if err != nil {
		return nil, err
	}
	return &MarkPriceStream{
		symbols: symbols,
		rate:    rate,
		quotes:  make(map[string]market.PriceQuote),
		stopCh:  make(chan struct{}),
	}, nil
}

func validateMarkPriceRate(rate time.Duration) error {
	switch rate {
	case 0, time.Second, 3 * time.Second:
		return nil
	default:
		return fmt.Errorf("invalid mark price rate %s: allowed values are 0, 1s, 3s", rate)
	}
}

func normalizeMarkPriceRate(rate time.Duration) (time.Duration, error) {
	if err := validateMarkPriceRate(rate); err != nil {
		return 0, err
	}
	if rate == 0 {
		return time.Second, nil
	}
	return rate, nil
}

func (s *MarkPriceStream) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("mark price stream is nil")
	}
	if s.running.Swap(true) {
		return nil
	}
	runID := s.nextRunID()
	s.setConnected(runID, false)
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	logger := logging.FromContext(baseCtx)
	streamCtx, cancel := context.WithCancel(baseCtx)
	streamCtx = logging.WithLogger(streamCtx, logger)
	stopCh := make(chan struct{})

	s.ctrlMu.Lock()
	s.stopCh = stopCh
	s.cancel = cancel
	s.ctrlMu.Unlock()

	go s.run(streamCtx, runID, stopCh)
	return nil
}

func (s *MarkPriceStream) Close() {
	if s == nil {
		return
	}
	if !s.running.Swap(false) {
		return
	}
	s.connected.Store(false)
	s.ctrlMu.Lock()
	cancel := s.cancel
	s.cancel = nil
	stopCh := s.stopCh
	s.ctrlMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if stopCh == nil {
		return
	}
	select {
	case <-stopCh:
	default:
		close(stopCh)
	}
}

func (s *MarkPriceStream) MarkPrice(ctx context.Context, symbol string) (market.PriceQuote, error) {
	if s == nil {
		return market.PriceQuote{}, market.ErrPriceUnavailable
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return market.PriceQuote{}, errors.New("symbol is required")
	}
	s.mu.RLock()
	quote, ok := s.quotes[symbol]
	s.mu.RUnlock()
	if !ok || quote.Price <= 0 {
		return market.PriceQuote{}, market.ErrPriceUnavailable
	}
	if !quoteIsFresh(quote, time.Now()) {
		return market.PriceQuote{}, market.ErrPriceUnavailable
	}
	return quote, nil
}

func (s *MarkPriceStream) run(ctx context.Context, runID uint64, stopCh <-chan struct{}) {
	logger := logging.FromContext(ctx).Named("market")
	defer logger.Info("mark price ws stopped")
	backoff := time.Second
	connectedOnce := false
	retrying := false
	for {
		if !s.running.Load() {
			s.setConnected(runID, false)
			return
		}
		doneC, stopC, err := s.serve(ctx)
		if err != nil {
			s.setConnected(runID, false)
			logger.Warn("mark price ws start failed", zap.Error(err))
			if connectedOnce {
				retrying = true
			}
			if !s.waitRetry(ctx, stopCh, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
			continue
		}
		if retrying {
			logger.Info("mark price ws reconnected")
		}
		s.setConnected(runID, true)
		connectedOnce = true
		retrying = false
		backoff = time.Second
		logger.Info("mark price ws connected")
		select {
		case <-ctx.Done():
			s.setConnected(runID, false)
			signalMarkPriceStop(doneC, stopC)
			return
		case <-stopCh:
			s.setConnected(runID, false)
			signalMarkPriceStop(doneC, stopC)
			return
		case <-doneC:
			s.setConnected(runID, false)
			if connectedOnce {
				retrying = true
			}
			if !s.waitRetry(ctx, stopCh, backoff) {
				return
			}
			backoff = nextBackoff(backoff)
		}
	}
}

func (s *MarkPriceStream) serve(ctx context.Context) (doneC, stopC chan struct{}, err error) {
	logger := logging.FromContext(ctx).Named("market")
	errHandler := func(err error) {
		if err != nil {
			logger.Warn("mark price ws error", zap.Error(err))
		}
	}
	handler := func(event *futures.WsMarkPriceEvent) {
		s.handleEvent(event)
	}
	if len(s.symbols) == 0 {
		allHandler := func(events futures.WsAllMarkPriceEvent) {
			for _, event := range events {
				s.handleEvent(event)
			}
		}
		return futures.WsAllMarkPriceServeWithRate(s.rate, allHandler, errHandler)
	}
	rates := make(map[string]time.Duration, len(s.symbols))
	for _, sym := range s.symbols {
		rates[sym] = s.rate
	}
	return futures.WsCombinedMarkPriceServeWithRate(rates, handler, errHandler)
}

func signalMarkPriceStop(doneC, stopC chan struct{}) bool {
	if stopC == nil {
		return false
	}
	if doneC != nil {
		select {
		case <-doneC:
			return false
		default:
		}
	}
	close(stopC)
	return true
}

func (s *MarkPriceStream) handleEvent(event *futures.WsMarkPriceEvent) {
	if event == nil {
		return
	}
	price, ok := parseutil.FloatStringOK(event.MarkPrice)
	if !ok || price <= 0 {
		return
	}
	symbol := strings.ToUpper(strings.TrimSpace(event.Symbol))
	if symbol == "" {
		return
	}
	eventTime := time.Now()
	if event.Time > 0 {
		eventTime = time.UnixMilli(event.Time)
	}
	quote := market.PriceQuote{
		Symbol:    symbol,
		Price:     price,
		Timestamp: event.Time,
		Source:    "binance_mark_ws",
	}
	s.mu.Lock()
	prev, ok := s.quotes[symbol]
	if ok && prev.Price > 0 && quoteIsFresh(prev, eventTime) {
		change := math.Abs(price-prev.Price) / prev.Price
		if change > defaultMarkPriceSpikePct {
			s.mu.Unlock()
			logging.L().Named("market").Warn("mark price spike rejected",
				zap.String("symbol", symbol),
				zap.Float64("prev_price", prev.Price),
				zap.Float64("new_price", price),
				zap.Float64("change_pct", change*100),
			)
			return
		}
	}
	s.quotes[symbol] = quote
	s.mu.Unlock()
}

func (s *MarkPriceStream) waitRetry(ctx context.Context, stopCh <-chan struct{}, backoff time.Duration) bool {
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-stopCh:
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(backoff time.Duration) time.Duration {
	next := backoff + time.Second
	if next > 5*time.Second {
		next = 5 * time.Second
	}
	jitterBound := next / 5
	if jitterBound <= 0 {
		return next
	}
	jitter := time.Duration(rand.Int63n(int64(jitterBound)))
	if rand.Intn(2) == 0 {
		return next + jitter
	}
	if next > jitter {
		return next - jitter
	}
	return next
}

func quoteIsFresh(quote market.PriceQuote, now time.Time) bool {
	if quote.Timestamp <= 0 {
		return false
	}
	return now.Sub(time.UnixMilli(quote.Timestamp)) <= defaultMarkPriceMaxAge
}

// StreamStatus returns the current stream health for a symbol.
// Implements market.PriceStreamInspector.
func (s *MarkPriceStream) StreamStatus(symbol string) (market.StreamStatus, bool) {
	if s == nil {
		return market.StreamStatus{}, false
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return market.StreamStatus{}, false
	}
	now := time.Now()
	s.mu.RLock()
	quote, ok := s.quotes[symbol]
	s.mu.RUnlock()
	if !ok || quote.Price <= 0 || quote.Timestamp <= 0 {
		return market.StreamStatus{}, false
	}

	status := market.StreamStatus{
		Symbol:    symbol,
		Source:    "binance_mark_price_ws",
		Connected: s.connected.Load(),
	}
	ts := time.UnixMilli(quote.Timestamp)
	status.LastPrice = quote.Price
	status.LastPriceTS = ts
	status.AgeMs = now.Sub(ts).Milliseconds()
	status.Fresh = quoteIsFresh(quote, now)
	return status, true
}

func normalizeSymbols(symbols []string) []string {
	if len(symbols) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(symbols))
	out := make([]string, 0, len(symbols))
	for _, sym := range symbols {
		sym = strings.ToUpper(strings.TrimSpace(sym))
		if sym == "" {
			continue
		}
		if _, ok := seen[sym]; ok {
			continue
		}
		seen[sym] = struct{}{}
		out = append(out, sym)
	}
	return out
}

func (s *MarkPriceStream) nextRunID() uint64 {
	if s == nil {
		return 0
	}
	return s.runID.Add(1)
}

func (s *MarkPriceStream) setConnected(runID uint64, connected bool) {
	if s == nil || runID == 0 {
		return
	}
	if s.runID.Load() != runID {
		return
	}
	s.connected.Store(connected)
}
