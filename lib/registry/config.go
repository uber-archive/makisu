//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

// Config contains registry client configuration.
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/uber/makisu/lib/registry/security"
	"github.com/uber/makisu/lib/utils"
	"github.com/uber/makisu/lib/utils/httputil"
)

// ConfigurationMap is a global variable that maps registry name to config.
var ConfigurationMap = Map{}

// DefaultDockerHubConfiguration contains docker hub registry configuration.
var DefaultDockerHubConfiguration = Config{
	Security: security.Config{
		TLS:       &httputil.TLSConfig{},
		BasicAuth: &security.BasicAuthConfig{}, // DockerHub requires empty username and password for public repositories.
	}}

// Map contains a map of registry config.
type Map map[string]RepositoryMap

// RepositoryMap contains a map of repo config. Repo name can be a regex.
type RepositoryMap map[string]Config

// Config contains docker registry client configuration.
type Config struct {
	Concurrency   int           `yaml:"concurrency" json:"concurrency"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	Retries       int           `yaml:"retries" json:"retries"`
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`
	RetryBackoff  float64       `yaml:"retry_backoff" json:"retry_backoff"`
	PushRate      float64       `yaml:"push_rate" json:"push_rate"`
	// If not specify, a default chunk size will be used.
	// Set it to -1 to turn off chunk upload.
	// NOTE: gcr and ecr do not support chunked upload.
	PushChunk int64           `yaml:"push_chunk" json:"push_chunk"`
	Security  security.Config `yaml:"security" json:"security"`
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
		c.Retries = 4
	}
	if c.RetryInterval == 0 {
		c.RetryInterval = 500 * time.Millisecond
	}
	if c.RetryBackoff == 0 {
		c.RetryBackoff = 2
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

func (c *Config) sendRetry() httputil.SendOption {
	return httputil.SendRetry(
		httputil.RetryMax(c.Retries),
		httputil.RetryInterval(c.RetryInterval),
		httputil.RetryBackoff(c.RetryBackoff))
}

// UpdateGlobalConfig updates the global registry config given either:
// - a JSON string of the configuration
// - a path to a YAML file
func UpdateGlobalConfig(registryConfig string) error {
	config := make(Map)
	if utils.IsValidJSON([]byte(registryConfig)) {
		if err := json.Unmarshal([]byte(registryConfig), &config); err != nil {
			return fmt.Errorf("unmarshal registry config: %s", err)
		}
	} else {
		data, err := ioutil.ReadFile(registryConfig)
		if err != nil {
			return fmt.Errorf("read registry config: %s", err)
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("unmarshal registry config: %s", err)
		}
	}

	for reg, repoConfig := range config {
		if _, ok := ConfigurationMap[reg]; !ok {
			ConfigurationMap[reg] = make(RepositoryMap)
		}
		for repo, config := range repoConfig {
			ConfigurationMap[reg][repo] = config
		}
	}
	return nil
}
