package httpclient

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	defaultRetries = 2
	defaultBackoff = 500 * time.Millisecond
)

var (
	defaultClient     *Client
	defaultClientOnce sync.Once
)

// Client wraps http.Client with retry and timeout helpers.
type Client struct {
	client  *http.Client
	retries int
	backoff time.Duration
}

// New creates a client with provided dialer. Nil client results in one with sensible defaults.
func New(client *http.Client, retries int, backoff time.Duration) *Client {
	if client == nil {
		client = &http.Client{
			Timeout: defaultTimeout,
		}
	} else if client.Timeout == 0 {
		client.Timeout = defaultTimeout
	}

	if retries < 0 {
		retries = defaultRetries
	}
	if backoff <= 0 {
		backoff = defaultBackoff
	}

	return &Client{
		client:  client,
		retries: retries,
		backoff: backoff,
	}
}

// Default returns a process-wide http client instance.
func Default() *Client {
	defaultClientOnce.Do(func() {
		timeout, retries, backoff := loadEnvConfig()
		defaultClient = New(&http.Client{Timeout: timeout}, retries, backoff)
	})
	return defaultClient
}

// Do executes the given request honoring retries and context.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	attempts := c.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		clonedReq := req.Clone(ctx)

		resp, err := c.client.Do(clonedReq)
		if err != nil {
			if attempt == attempts-1 || !isRetryableError(err) {
				return nil, err
			}
		} else {
			if resp.StatusCode >= 500 && attempt < attempts-1 {
				resp.Body.Close()
			} else {
				return resp, nil
			}
		}

		select {
		case <-time.After(c.backoff * time.Duration(attempt+1)):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, errors.New("httpclient: exhausted retries")
}

func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func loadEnvConfig() (time.Duration, int, time.Duration) {
	timeout := defaultTimeout
	if v := os.Getenv("BINENV_HTTP_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	retries := defaultRetries
	if v := os.Getenv("BINENV_HTTP_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			retries = n
		}
	}

	backoff := defaultBackoff
	if v := os.Getenv("BINENV_HTTP_BACKOFF"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			backoff = parsed
		}
	}

	return timeout, retries, backoff
}
