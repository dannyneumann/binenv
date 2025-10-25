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

func TestGitlabReleaseGetHandlesRateLimit(t *testing.T) {
	t.Parallel()
	var calls int
	responder := mockRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		headers := http.Header{}
		if calls == 1 {
			headers.Set("Ratelimit-Remaining", "0")
			headers.Set("Ratelimit-Limit", "60")
			headers.Set("Ratelimit-Reset", "2000000000")
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     headers,
				Body:       io.NopCloser(strings.NewReader("{}")),
			}, nil
		}
		headers.Set("Ratelimit-Remaining", "10")
		body := `[{"tag_name":"v1.2.3"}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	logger := zerolog.New(io.Discard)
	ctx := logger.WithContext(context.Background())

	lister := &GitlabRelease{
		url:    "https://gitlab.example.org/api/v4/projects/1/releases",
		client: httpclient.New(&http.Client{Transport: responder}, 0, 0),
	}

	_, err := lister.Get(ctx)
	if err == nil {
		t.Fatalf("expected rate limit error, got nil")
	}
}

func TestGitlabReleaseFiltersPrefixAndExclude(t *testing.T) {
	t.Parallel()
	responder := mockRoundTripper(func(r *http.Request) (*http.Response, error) {
		headers := http.Header{}
		headers.Set("Ratelimit-Remaining", "10")
		body := `[{"tag_name":"rel-1.0.0"},{"tag_name":"rel-1.0.0-rc1"}]`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     headers,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})

	logger := zerolog.New(io.Discard)
	ctx := logger.WithContext(context.Background())

	lister := &GitlabRelease{
		url:         "https://gitlab.example.org/api/v4/projects/1/releases",
		prefix:      "rel-",
		exclude:     "rc",
		versionFrom: "tag_name",
		client:      httpclient.New(&http.Client{Transport: responder}, 0, 0),
	}

	versions, err := lister.Get(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 1 || versions[0] != "1.0.0" {
		t.Fatalf("unexpected versions: %v", versions)
	}
}
