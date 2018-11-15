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

package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall"
	"time"
)

const maxUnixSocketPathSize = len(syscall.RawSockaddrUnix{}.Path)
const defaultTimeout = 32 * time.Second
const clientDir = "docker"
const perm = 0755

// DockerClient connects to docker daemon socket
type DockerClient struct {
	rootDir    string      // root directory
	version    string      // docker version
	host       string      // host that client connects to
	scheme     string      // http/https
	userHeader http.Header // user configured header

	addr     string       // client address
	protocol string       // unix
	basePath string       // base part of the url
	client   *http.Client // opens http.transport
}

// NewDockerClient creates a new DockerClient
func NewDockerClient(
	sandboxDir string, host, scheme, version string, headers http.Header) (*DockerClient, error) {
	rootDir := path.Join(sandboxDir, clientDir)
	err := os.MkdirAll(rootDir, perm)
	if err != nil {
		return nil, err
	}
	protocol, addr, basePath, err := parseHost(host)
	if err != nil {
		return nil, err
	}

	transport := new(http.Transport)
	configureTransport(transport, protocol, addr)
	client := &http.Client{
		Transport: transport,
	}

	return &DockerClient{
		rootDir:    rootDir,
		scheme:     scheme,
		host:       host,
		version:    version,
		userHeader: headers,
		protocol:   protocol,
		addr:       addr,
		basePath:   basePath,
		client:     client,
	}, nil
}

// ImageTarLoad calls `docker load` on an image tar
func (cli *DockerClient) ImageTarLoad(ctx context.Context, input io.Reader) error {
	v := url.Values{}
	v.Set("quiet", "1")
	headers := map[string][]string{"Content-Type": {"application/x-tar"}}
	return cli.post(ctx, "/images/load", v, input, headers, false)
}

func parseHost(host string) (string, string, string, error) {
	strs := strings.SplitN(host, "://", 2)
	if len(strs) == 1 {
		return "", "", "", fmt.Errorf("unable to parse docker host `%s`", host)
	}

	var basePath string
	protocol, addr := strs[0], strs[1]
	if protocol == "tcp" {
		parsed, err := url.Parse("tcp://" + addr)
		if err != nil {
			return "", "", "", err
		}
		addr = parsed.Host
		basePath = parsed.Path
	}
	return protocol, addr, basePath, nil
}

func configureTransport(tr *http.Transport, protocol, addr string) error {
	switch protocol {
	case "unix":
		if len(addr) > maxUnixSocketPathSize {
			return fmt.Errorf("Unix socket path %q is too long", addr)
		}

		tr.DisableCompression = true
		tr.Dial = func(_, _ string) (net.Conn, error) {
			return net.DialTimeout(protocol, addr, defaultTimeout)
		}
		return nil
	}

	return fmt.Errorf("Protocol %s not supported", protocol)
}
