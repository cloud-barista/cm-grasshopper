package common

import (
	"context"
	"github.com/jollaman999/utils/logger"
	"io"
	"net/http"
)

func GetHTTPRequest(URL string) ([]byte, error) {
	ctx := context.Background()
	client := &http.Client{}

	logger.Println(logger.DEBUG, false, "GetHTTPRequest: Requesting URL='"+URL+"'")

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
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

	return responseBody, nil
}
