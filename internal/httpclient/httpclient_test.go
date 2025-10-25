package httpclient

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockTransport func(*http.Request) (*http.Response, error)

func (m mockTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return m(r)
}

func TestClientRetriesOnServerError(t *testing.T) {
	t.Parallel()
	var calls int
	rt := mockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	client := New(&http.Client{Transport: rt}, 1, 0)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success after retry, got error %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestClientRetriesOnTimeout(t *testing.T) {
	t.Parallel()
	var calls int
	rt := mockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls < 2 {
			return nil, timeoutError{}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
		}, nil
	})

	client := New(&http.Client{Transport: rt}, 2, time.Millisecond)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	resp, err := client.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if calls < 2 {
		t.Fatalf("expected retries to be attempted")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestClientRespectsContextCancellation(t *testing.T) {
	t.Parallel()
	rt := mockTransport(func(r *http.Request) (*http.Response, error) {
		select {
		case <-time.After(50 * time.Millisecond):
			return nil, errors.New("unreachable")
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})

	client := New(&http.Client{Transport: rt}, 2, time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	_, err := client.Do(ctx, req)
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestClientDoesNotRetryNonRetryableError(t *testing.T) {
	t.Parallel()
	var calls int
	rt := mockTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, &net.DNSError{IsTimeout: false}
	})

	client := New(&http.Client{Transport: rt}, 3, time.Millisecond)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	_, err := client.Do(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected non-retryable error to stop retries, got %d calls", calls)
	}
}
