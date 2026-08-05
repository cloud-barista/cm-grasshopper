package common

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/jollaman999/utils/logger"
)

func GetHTTPRequest(URL string, username string, password string) ([]byte, error) {
	ctx := context.Background()
	client := &http.Client{}

	logger.Println(logger.DEBUG, false, "GetHTTPRequest: Requesting URL='"+URL+"'")

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
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
