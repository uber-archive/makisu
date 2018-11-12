package cli

import (
	"context"
	"fmt"
	"net/url"
)

// ImagePull calls `docker pull` on an image
func (cli *DockerClient) ImagePull(ctx context.Context, registry, repo, tag string) error {
	v := url.Values{}
	fromImage := repo
	if registry != "" {
		fromImage = fmt.Sprintf("%s/%s", registry, repo)
	}
	v.Set("fromImage", fromImage)
	v.Set("tag", tag)
	headers := map[string][]string{"X-Registry-Auth": {""}}
	return cli.post(ctx, "/images/create", v, nil, headers, true)
}
