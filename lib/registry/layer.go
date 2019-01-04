package registry

import (
	"os"

	"github.com/uber/makisu/lib/docker/image"
)

// LayerClient is the interface that exposes the direct interaction with image layers.
type LayerClient interface {
	PullLayer(layerDigest image.Digest) (os.FileInfo, error)
	PushLayer(layerDigest image.Digest) error
}
