package app

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/devops-works/binenv/internal/httpclient"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestAppWithClient(t *testing.T, client *http.Client) *App {
	t.Helper()
	a := &App{
		cache:      make(map[string][]string),
		logger:     zerolog.Nop(),
		httpClient: httpclient.New(client, 0, time.Millisecond),
	}
	return a
}

func TestDownloadOrFallbackRemoteSuccess(t *testing.T) {
	t.Parallel()
	body := []byte("remote")
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		}, nil
	})
	app := newTestAppWithClient(t, &http.Client{Transport: rt})

	data, usedFallback, err := app.downloadOrFallback("http://example.com", "resource", []byte("fallback"))
	if err != nil {
		t.Fatalf("downloadOrFallback returned error: %v", err)
	}
	if usedFallback {
		t.Fatalf("expected remote data to be used without fallback")
	}
	if !bytes.Equal(data, body) {
		t.Fatalf("expected %q, got %q", body, data)
	}
}

func TestDownloadOrFallbackUsesFallbackOnError(t *testing.T) {
	t.Parallel()
	fallback := []byte("fallback")
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})
	app := newTestAppWithClient(t, &http.Client{Transport: rt})

	data, usedFallback, err := app.downloadOrFallback("http://example.com", "resource", fallback)
	if err != nil {
		t.Fatalf("downloadOrFallback returned error: %v", err)
	}
	if !usedFallback {
		t.Fatalf("expected fallback to be used")
	}
	if !bytes.Equal(data, fallback) {
		t.Fatalf("expected fallback data, got %q", data)
	}
}

func TestDownloadOrFallbackErrorWithoutFallback(t *testing.T) {
	t.Parallel()
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})
	app := newTestAppWithClient(t, &http.Client{Transport: rt})

	if _, _, err := app.downloadOrFallback("http://example.com", "resource", nil); err == nil {
		t.Fatalf("expected error when no fallback data is available")
	}
}

func TestLoadCacheData(t *testing.T) {
	t.Parallel()
	a := &App{}
	count, err := a.loadCacheData([]byte(`{"terraform":["1.0.0"]}`))
	if err != nil {
		t.Fatalf("loadCacheData returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if got := a.cache["terraform"][0]; got != "1.0.0" {
		t.Fatalf("unexpected cache content: %v", got)
	}
}

func TestLoadDistributionsData(t *testing.T) {
	t.Parallel()
	yamlData := []byte("sources:\n  terraform:\n    description: tool\n    url: https://example.com")
	a := &App{}
	count, err := a.loadDistributionsData(yamlData)
	if err != nil {
		t.Fatalf("loadDistributionsData returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 distribution, got %d", count)
	}
	if _, ok := a.def.Sources["terraform"]; !ok {
		t.Fatalf("expected terraform to be present in sources")
	}
}

func TestShouldDisableProgress(t *testing.T) {
	t.Run("explicit true", func(t *testing.T) {
		t.Setenv(disableProgressEnv, "true")
		if !shouldDisableProgress() {
			t.Fatalf("expected progress to be disabled")
		}
	})

	t.Run("invalid but set", func(t *testing.T) {
		t.Setenv(disableProgressEnv, "invalid")
		if !shouldDisableProgress() {
			t.Fatalf("expected progress to be disabled when env set to invalid value")
		}
	})

	t.Run("ci default", func(t *testing.T) {
		t.Setenv(disableProgressEnv, "")
		t.Setenv(ciEnv, "true")
		if !shouldDisableProgress() {
			t.Fatalf("expected progress to be disabled in CI")
		}
	})

	t.Run("default false", func(t *testing.T) {
		t.Setenv(disableProgressEnv, "")
		t.Setenv(ciEnv, "")
		if shouldDisableProgress() {
			t.Fatalf("expected progress to be enabled by default")
		}
	})
}
