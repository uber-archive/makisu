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

package security

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/utils/httputil"

	"github.com/docker/engine-api/types"
)

// Config contains tls and basic auth configuration.
type Config struct {
	TLS       *httputil.TLSConfig `yaml:"tls"`
	BasicAuth *types.AuthConfig   `yaml:"basic"`
}

// ApplyDefaults applies default configuration.
func (c Config) ApplyDefaults() Config {
	if c.TLS == nil {
		c.TLS = &httputil.TLSConfig{}
	}
	if c.TLS.CA.Cert.Path == "" {
		c.TLS.CA.Cert.Path = pathutils.DefaultCACertsPath
	}
	return c
}

// GetHTTPOption returns httputil.Option based on the security configuration.
func (c Config) GetHTTPOption(addr, repo string) (httputil.SendOption, error) {
	var tlsClientConfig *tls.Config
	var err error
	if c.TLS != nil {
		tlsClientConfig, err = c.TLS.BuildClient()
		if err != nil {
			return nil, fmt.Errorf("build tls config: %s", err)
		}
		if c.BasicAuth == nil {
			return httputil.SendTLS(tlsClientConfig), nil
		}
	}
	if c.BasicAuth != nil {
		tr := http.DefaultTransport.(*http.Transport)
		tr.TLSClientConfig = tlsClientConfig // If tlsClientConfig is nil, default is used.
		rt, err := BasicAuthTransport(addr, repo, tr, *c.BasicAuth)
		if err != nil {
			return nil, fmt.Errorf("basic auth: %s", err)
		}
		return httputil.SendTLSTransport(rt), nil
	}
	return httputil.SendNoop(), nil
}
