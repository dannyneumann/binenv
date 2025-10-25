package list

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/devops-works/binenv/internal/httpclient"
)

type mockRoundTripper func(*http.Request) (*http.Response, error)

func (m mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return m(r)
}

func TestGithubReleaseGetHandlesRateLimit(t *testing.T) {
	t.Parallel()
	var calls int
	responder := mockRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		headers := http.Header{}
		if calls == 1 {
			headers.Set("X-Ratelimit-Remaining", "0")
			headers.Set("X-Ratelimit-Limit", "60")
			headers.Set("X-Ratelimit-Reset", "2000000000")
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     headers,
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		}
		headers.Set("X-Ratelimit-Remaining", "10")
		body := `[{"tag_name":"v1.0.0"}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	logger := zerolog.New(io.Discard)
	ctx := logger.WithContext(context.Background())

	lister := &GithubRelease{
		url:    "https://api.github.com/repos/example/tool/releases",
		client: httpclient.New(&http.Client{Transport: responder}, 0, 0),
	}

	_, err := lister.Get(ctx)
	if err == nil {
		t.Fatalf("expected error due to rate limit, got nil")
	}
}

func TestGithubReleaseGetFiltersPrefix(t *testing.T) {
	t.Parallel()
	responder := mockRoundTripper(func(r *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"tool-v1.0.0"},{"tag_name":"tool-v0.9.0"}]`
		headers := http.Header{}
		headers.Set("X-Ratelimit-Remaining", "10")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	logger := zerolog.New(io.Discard)
	ctx := logger.WithContext(context.Background())

	lister := &GithubRelease{
		url:    "https://api.github.com/repos/example/tool/releases",
		prefix: "tool-",
		client: httpclient.New(&http.Client{Transport: responder}, 0, 0),
	}

	versions, err := lister.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 || versions[0] != "v1.0.0" {
		t.Fatalf("unexpected versions: %v", versions)
	}
}

func TestGithubReleaseGetFiltersExclude(t *testing.T) {
	t.Parallel()
	responder := mockRoundTripper(func(r *http.Request) (*http.Response, error) {
		body := `[{"tag_name":"v1.0.0"},{"tag_name":"v1.0.0-rc1"}]`
		headers := http.Header{}
		headers.Set("X-Ratelimit-Remaining", "10")
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	logger := zerolog.New(io.Discard)
	ctx := logger.WithContext(context.Background())

	lister := &GithubRelease{
		url:     "https://api.github.com/repos/example/tool/releases",
		exclude: "rc",
		client:  httpclient.New(&http.Client{Transport: responder}, 0, 0),
	}

	versions, err := lister.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 1 || versions[0] != "v1.0.0" {
		t.Fatalf("expected stable version only, got %v", versions)
	}
}
