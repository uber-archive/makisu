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
