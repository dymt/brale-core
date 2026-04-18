package execution

import (
	"context"
	"errors"
	"net/http"
	"time"

	"brale-core/internal/pkg/logging"

	"github.com/hashicorp/go-retryablehttp"
)

func newFreqtradeRetryClient(base *http.Client) *retryablehttp.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 200 * time.Millisecond
	client.RetryWaitMax = 2 * time.Second
	client.Logger = logging.RetryableHTTPLogger{Logger: logging.L().Named("freqtrade-retry")}
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return false, err
			}
			return true, nil
		}
		if resp == nil {
			return false, nil
		}
		switch {
		case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
			return true, nil
		case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
			return false, nil
		}
		return false, nil
	}
	if base != nil {
		client.HTTPClient = base
	}
	return client
}
