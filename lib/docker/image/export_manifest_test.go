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

func TestExportManifest(t *testing.T) {
	manifest, _, err := UnmarshalDistributionManifest(MediaTypeManifest, []byte(busyboxDistManifest))
	require.NoError(t, err)
	expManifest := NewExportManifestFromDistribution(Name{}, manifest)
	require.Equal(t, 1, len(expManifest.Layers))
	layer := expManifest.Layers[0]
	require.Equal(t, "393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b", layer.ID())
	require.Equal(t, "393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b/layer.tar", layer.String())
	require.Equal(t, "411a417c1f6ef5b93fac71c92276013f45762dde0bb36a80a6148ca114d1b0fa", expManifest.Config.ID())
}
