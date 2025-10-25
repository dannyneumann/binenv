package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/schollz/progressbar/v3"

	"github.com/devops-works/binenv/internal/httpclient"
	"github.com/devops-works/binenv/internal/mapping"
	"github.com/devops-works/binenv/internal/tpl"
)

// Download handles direct binary releases
type Download struct {
	url     string
	headers map[string]string
	client  *httpclient.Client
}

// Fetch gets the package and returns location of downloaded file
func (d Download) Fetch(ctx context.Context, dist, v string, mapper mapping.Mapper) (string, error) {
	logger := zerolog.Ctx(ctx).With().Str("func", "Download.Fetch").Logger()

	args := tpl.New(v, mapper)

	url, err := args.Render(d.url)
	if err != nil {
		return "", err
	}

	logger.Debug().Msgf("fetching version %q for arch %q and OS %q at %s", v, runtime.GOARCH, runtime.GOOS, url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	for k, v := range d.headers {
		req.Header.Add(k, v)
	}

	client := d.client
	if client == nil {
		client = httpclient.Default()
	}

	resp, err := client.Do(ctx, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to download binary at %s: %s", url, resp.Status)
	}

	tmpfile, err := os.CreateTemp("", v)
	if err != nil {
		logger.Fatal().Err(err)
	}

	defer tmpfile.Close()

	writer := io.Writer(tmpfile)
	if resp.ContentLength > 0 {
		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			fmt.Sprintf("fetching %s version %s", dist, v),
		)
		writer = io.MultiWriter(tmpfile, bar)
	}

	_, err = io.Copy(writer, resp.Body)

	return tmpfile.Name(), err
}
