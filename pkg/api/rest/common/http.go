package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jollaman999/utils/logger"
)

// Retry tuning for rate-limited / transiently-unavailable upstreams (e.g.
// cb-tumblebug rate-limits some routes at 2 req/s). On 429/503 we back off and
// retry rather than failing, so callers keep working under the limit.
const (
	httpMaxRetries    = 5
	httpRetryBaseWait = 500 * time.Millisecond
	httpRetryMaxWait  = 8 * time.Second
)

func GetHTTPRequest(URL string, username string, password string) ([]byte, error) {
	ctx := context.Background()
	client := &http.Client{}

	logger.Println(logger.DEBUG, false, "GetHTTPRequest: Requesting URL='"+URL+"'")

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodGet, URL, nil)
		if err != nil {
			return nil, err
		}

		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}

		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		responseBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		// Rate-limited (429) or temporarily unavailable (503): don't fail — wait
		// and retry so the caller succeeds once tokens refill.
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < httpMaxRetries {
				wait := retryWait(attempt, resp.Header.Get("Retry-After"))
				logger.Println(logger.DEBUG, false, fmt.Sprintf(
					"GetHTTPRequest: HTTP %d from '%s', retrying in %s (attempt %d/%d)",
					resp.StatusCode, URL, wait, attempt+1, httpMaxRetries))
				time.Sleep(wait)
				continue
			}
		}

		// Surface non-2xx responses as errors. Previously the body was returned
		// regardless of status, so a rejection like cb-tumblebug's rate-limit
		// (429 {"message":"rate limit exceeded"}) unmarshalled into an empty struct
		// and looked like a valid-but-empty result to the caller.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			snippet := string(responseBody)
			if len(snippet) > 300 {
				snippet = snippet[:300]
			}
			return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, URL, snippet)
		}

		return responseBody, nil
	}
}

// retryWait returns how long to wait before the next attempt. It honors a
// Retry-After header (seconds) when present, otherwise uses capped exponential
// backoff starting at httpRetryBaseWait.
func retryWait(attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
			wait := time.Duration(secs) * time.Second
			if wait > httpRetryMaxWait {
				wait = httpRetryMaxWait
			}
			return wait
		}
	}

	wait := httpRetryBaseWait << attempt // 500ms, 1s, 2s, 4s, ...
	if wait > httpRetryMaxWait {
		wait = httpRetryMaxWait
	}
	return wait
}
