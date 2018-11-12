package registry

// Config contains registry client configuration.
import (
	"time"

	"github.com/docker/engine-api/types"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/registry/security"
	"github.com/uber/makisu/lib/utils/httputil"
)

// ConfigurationMap is a global variable that maps registry name to config.
var ConfigurationMap = Map{
	image.DockerHubRegistry: RepositoryMap{"library/*": DefaultDockerHubConfiguration},
}

// DefaultDockerHubConfiguration contains docker hub registry configuration.
var DefaultDockerHubConfiguration = Config{
	Security: security.Config{
		TLS: &httputil.TLSConfig{
			Client: httputil.X509Pair{Enabled: true}},
		BasicAuth: &types.AuthConfig{},
	}}

// Map contains a map of registry config.
type Map map[string]RepositoryMap

// RepositoryMap contains a map of repo config. Repo name can be a regex.
type RepositoryMap map[string]Config

// Config contains docker registry client configuration.
type Config struct {
	Concurrency int           `yaml:"concurrency"`
	Timeout     time.Duration `yaml:"timeout"`
	Retries     int           `yaml:"retries"`
	PushRate    float64       `yaml:"push_rate"`
	// If not specify, a default chunk size will be used.
	// Set it to -1 to turn off chunk upload.
	// NOTE: gcr does not support chunked upload.
	PushChunk int64           `yaml:"push_chunk"`
	Security  security.Config `yaml:"security"`
}

func (c Config) applyDefaults() Config {
	if c.Concurrency == 0 {
		c.Concurrency = 3
	}
	// TODO: Decrease the timeout. 10 mins is too long.
	if c.Timeout == 0 {
		c.Timeout = 600 * time.Second
	}
	if c.Retries == 0 {
		c.Retries = 3
	}
	if c.PushRate == 0 {
		c.PushRate = 100 * 1024 * 1024 // 100 MB/s
	}
	if c.PushChunk == 0 {
		c.PushChunk = 50 * 1024 * 1024 // 50 MB
	}
	c.Security = c.Security.ApplyDefaults()
	return c
}
