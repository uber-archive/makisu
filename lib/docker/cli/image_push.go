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
	"encoding/base64"
	"fmt"
	"net/url"
)

// ImagePush calls `docker push` on an image
func (cli *DockerClient) ImagePush(ctx context.Context, registry, repo, tag string) error {
	v := url.Values{}
	image := repo
	if registry != "" {
		image = fmt.Sprintf("%s/%s", registry, repo)
	}
	v.Set("tag", tag)

	headers := map[string][]string{
		"X-Registry-Auth": {base64.URLEncoding.EncodeToString([]byte("{\"username\":\"\",\"password\":\"\", \"auth\":\"\",\"email\":\"\"}"))},
	}
	return cli.post(ctx, fmt.Sprintf("/images/%s/push", image), v, nil, headers, true)
}
