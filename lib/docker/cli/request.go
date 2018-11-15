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
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/uber/makisu/lib/log"

	"golang.org/x/net/context/ctxhttp"
)

func (cli *DockerClient) getAPIPath(p string, query url.Values) string {
	var apiPath string
	if cli.version != "" {
		v := strings.TrimPrefix(cli.version, "v")
		apiPath = fmt.Sprintf("%s/v%s%s", cli.basePath, v, p)
	} else {
		apiPath = fmt.Sprintf("%s%s", cli.basePath, p)
	}

	u := &url.URL{
		Path: apiPath,
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

// post sends post request
func (cli *DockerClient) post(ctx context.Context, url string, query url.Values, body io.Reader, header http.Header, streamRespBody bool) error {
	if body == nil {
		body = bytes.NewReader([]byte{})
	}
	resp, err := cli.doRequest(ctx, "POST", cli.getAPIPath(url, query), body, header)
	if err != nil {
		return fmt.Errorf("post request: %s", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		errMsg, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read error resp: %s", err)
		}
		return fmt.Errorf("Error posting to %s: code %d, err: %s", url, resp.StatusCode, errMsg)
	}

	// Docker daemon returns 200 before complete push
	// it closes resp.Body after it finishes
	if streamRespBody {
		log.Debugf("Streaming resp body for %s", url)
		progress, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read resp body: %s", err)
		}
		log.Debugf("%s", progress)
	}

	return nil
}

func (cli *DockerClient) doRequest(
	ctx context.Context,
	method string,
	url string,
	body io.Reader,
	header http.Header) (*http.Response, error) {

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = cli.mergeHeader(cli.userHeader, header)
	req.Host = "docker"
	req.URL.Host = cli.addr
	req.URL.Scheme = cli.scheme

	return ctxhttp.Do(ctx, cli.client, req)
}

func (cli *DockerClient) mergeHeader(header http.Header, overwriteHeader http.Header) http.Header {
	resultHeader := make(http.Header)

	for k, v := range header {
		resultHeader[k] = v
	}

	for k, v := range overwriteHeader {
		resultHeader[k] = v
	}
	return resultHeader
}
