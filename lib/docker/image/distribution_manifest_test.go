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

const busyboxDistManifest = `{
   "schemaVersion":2,
   "mediaType":"application/vnd.docker.distribution.manifest.v2+json",
   "config":{
      "mediaType":"application/vnd.docker.container.image.v1+json",
      "size":1346,
      "digest":"411a417c1f6ef5b93fac71c92276013f45762dde0bb36a80a6148ca114d1b0fa"
   },
   "layers":[
      {
         "mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip",
         "size":1308672,
         "digest":"393ccd5c4dd90344c9d725125e13f636ce0087c62f5ca89050faaacbb9e3ed5b"
      }
   ]
}`

func TestUnmarshalDistributionManifest(t *testing.T) {
	require := require.New(t)

	manifest, _, err := UnmarshalDistributionManifest(
		MediaTypeManifest, []byte(busyboxDistManifest))
	require.NoError(err)
	require.Equal(1, len(manifest.GetLayerDigests()))
}
