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

	name, err = ParseNameForPull("uber-usi/dockermover")
	require.NoError(err)
	require.Equal(DockerHubRegistry, name.registry)
	require.Equal(name.GetRepository(), "uber-usi/dockermover")
	require.Equal(name.GetTag(), "latest")
	require.True(name.IsValid())
	require.Equal("index.docker.io/uber-usi/dockermover:latest", name.String())

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
