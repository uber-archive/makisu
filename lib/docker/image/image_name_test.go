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

package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseImageName(t *testing.T) {
	require := require.New(t)

	name := MustParseName("127.0.0.1:15055/uber-usi/dockermover:sjc1-prod-0000000001")
	require.Equal(name.GetRegistry(), "127.0.0.1:15055")
	require.Equal(name.GetRepository(), "uber-usi/dockermover")
	require.Equal(name.GetTag(), "sjc1-prod-0000000001")
	require.True(name.IsValid())
	require.Equal("127.0.0.1:15055/uber-usi/dockermover:sjc1-prod-0000000001", name.String())

	name, err := ParseNameForPull("docker-registry.pit-irn.uberatc.net/uber-usi/dockermover")
	require.NoError(err)
	require.Equal(name.GetRegistry(), "docker-registry.pit-irn.uberatc.net")
	require.Equal(name.GetRepository(), "uber-usi/dockermover")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("docker-registry.pit-irn.uberatc.net/uber-usi/dockermover:latest", name.String())

	name, err = ParseNameForPull("127.0.0.1:5002/evanescence-golang-1:latest")
	require.NoError(err)
	require.Equal(name.GetRegistry(), "127.0.0.1:5002")
	require.Equal(name.GetRepository(), "evanescence-golang-1")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("127.0.0.1:5002/evanescence-golang-1:latest", name.String())

	name, err = ParseNameForPull("docker-registry01-sjc1:5055/uber-usi/haproxy-agent:sjc1-produ-0000000027")
	require.NoError(err)
	require.Equal(name.GetRegistry(), "docker-registry01-sjc1:5055")
	require.Equal(name.GetRepository(), "uber-usi/haproxy-agent")
	require.Equal(name.GetTag(), "sjc1-produ-0000000027")
	require.True(name.IsValid())
	require.Equal("docker-registry01-sjc1:5055/uber-usi/haproxy-agent:sjc1-produ-0000000027", name.String())

	name, err = ParseNameForPull("scratch")
	require.NoError(err)
	require.Equal("", name.GetRegistry())
	require.Equal(name.GetRepository(), "scratch")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("scratch:latest", name.String())

}

func TestParseImageNameForDockerHub(t *testing.T) {
	require := require.New(t)

	name, err := ParseNameForPull("wodby/php")
	require.NoError(err)
	require.Equal(DockerHubRegistry, name.registry)
	require.Equal(name.GetRepository(), "wodby/php")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("index.docker.io/wodby/php:latest", name.String())

	name, err = ParseNameForPull("index.docker.io/php")
	require.NoError(err)
	require.Equal(DockerHubRegistry, name.registry)
	require.Equal(name.GetRepository(), "library/php")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("index.docker.io/library/php:latest", name.String())

	name, err = ParseNameForPull("index.docker.io/wodby/php")
	require.NoError(err)
	require.Equal(DockerHubRegistry, name.registry)
	require.Equal(name.GetRepository(), "wodby/php")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("index.docker.io/wodby/php:latest", name.String())

	name, err = ParseNameForPull("debian:9")
	require.NoError(err)
	require.Equal(DockerHubRegistry, name.registry)
	require.Equal(name.GetRepository(), "library/debian")
	require.Equal(name.GetTag(), "9")
	require.True(name.IsValid())
	require.Equal("index.docker.io/library/debian:9", name.String())
}
