package image

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExportManifest(t *testing.T) {
	manifest, _, err := UnmarshalDistributionManifest(MediaTypeManifest, []byte(testManifest))
	require.NoError(t, err)
	expManifest := NewExportManifestFromDistribution(Name{}, manifest)
	require.Equal(t, 1, len(expManifest.Layers))
	layer := expManifest.Layers[0]
	require.Equal(t, "d660b1f15b9bfb8142f50b518156f2d364d9642fe05854538b060498e2f7928d", layer.ID())
	require.Equal(t, "d660b1f15b9bfb8142f50b518156f2d364d9642fe05854538b060498e2f7928d/layer.tar", layer.String())
	require.Equal(t, "79f4bda919894b2fe9a66f403337bdc0c547ac95183ec034a3a37869e17ee72e", expManifest.Config.ID())
}
