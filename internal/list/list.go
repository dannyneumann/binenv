package list

import (
	"context"

	"github.com/devops-works/binenv/internal/httpclient"
)

// Lister should return a list of available release versions
type Lister interface {
	Get(ctx context.Context) ([]string, error)
}

// List contains list definition
type List struct {
	Type        string `yaml:"type"`
	Prefix      string `yaml:"prefix"`
	Exclude     string `yaml:"exclude"` // exclude versions containing this regex
	VersionFrom string `yaml:"version_from"`
	URL         string `yaml:"url"`
	TokenEnv    string `yaml:"token_env"`
	Versions    []string
}

// Factory returns instances that comply to Lister interface
func (l List) Factory(client *httpclient.Client) Lister {
	if client == nil {
		client = httpclient.Default()
	}
	switch l.Type {
	case "github-releases":
		return GithubRelease{
			url:         l.URL,
			prefix:      l.Prefix,
			versionFrom: l.VersionFrom,
			exclude:     l.Exclude,
			client:      client,
		}
	case "gitlab-releases":
		return GitlabRelease{
			url:         l.URL,
			prefix:      l.Prefix,
			versionFrom: l.VersionFrom,
			exclude:     l.Exclude,
			tokenEnv:    l.TokenEnv,
			client:      client,
		}
	case "static":
		return Static{
			versions: l.Versions,
		}
	}
	return nil
}
