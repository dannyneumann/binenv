package distributions

import _ "embed"

var (
	//go:embed distributions.yaml
	DistributionsYAML []byte

	//go:embed cache.json
	CacheJSON []byte
)
