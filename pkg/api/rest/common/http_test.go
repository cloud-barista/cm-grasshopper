package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetHTTPRequestStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer srv.Close()

	url := srv.URL + "/path"
	_, err := GetHTTPRequest(url, "", "")

	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v (%T), want *HTTPStatusError", err, err)
	}
	if statusErr.StatusCode != http.StatusBadRequest || statusErr.URL != url || statusErr.Body != `{"error":"bad"}` {
		t.Errorf("statusErr = %+v", statusErr)
	}
	if want := `HTTP 400 from ` + url + `: {"error":"bad"}`; err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestHTTPStatusErrorTruncatesBody(t *testing.T) {
	err := &HTTPStatusError{StatusCode: 500, URL: "http://example.com/x", Body: strings.Repeat("a", 400)}
	if want := "HTTP 500 from http://example.com/x: " + strings.Repeat("a", 300); err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestGetHTTPRequestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	data, err := GetHTTPRequest(srv.URL, "", "")
	if err != nil || string(data) != "ok" {
		t.Errorf("data = %q, err = %v", data, err)
	}
}
