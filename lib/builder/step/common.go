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

package step

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io/ioutil"
	"os"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/stream"
	"github.com/uber/makisu/lib/tario"
)

// tarAndGzipDiffs tars and gzips files to a temporary location.
// It returns two digesters and the temporary file name.
func tarAndGzipDiffs(ctx *context.BuildContext, writeDiffs func(*tar.Writer) error) (
	gzipDigester hash.Hash, tarDigester hash.Hash, name string, err error) {

	tempGzipTar, err := ioutil.TempFile(ctx.ImageStore.SandboxDir, "layertar-")
	if err != nil {
		return nil, nil, "", fmt.Errorf("temp gzip tar file: %s", err)
	}
	defer tempGzipTar.Close()

	gzipDigester = sha256.New()
	tarDigester = sha256.New()

	gzipMulti := stream.NewConcurrentMultiWriter(tempGzipTar, gzipDigester)
	gzipper, err := tario.NewGzipWriter(gzipMulti)
	if err != nil {
		return nil, nil, "", fmt.Errorf("new gzip writer: %s", err)
	}
	defer gzipper.Close()

	multiWriter := stream.NewConcurrentMultiWriter(tarDigester, gzipper)
	tarWriter := tar.NewWriter(multiWriter)
	defer tarWriter.Close()

	if err := writeDiffs(tarWriter); err != nil {
		return nil, nil, "", fmt.Errorf("write diffs: %s", err)
	}

	return gzipDigester, tarDigester, tempGzipTar.Name(), nil
}

// commitLayer commits a layer by either scan or copy operations, depending on the context.
func commitLayer(ctx *context.BuildContext) ([]*image.DigestPair, error) {
	var writeDiffs func(w *tar.Writer) error
	if ctx.MustScan {
		writeDiffs = ctx.MemFS.AddLayerByScan
	} else if len(ctx.CopyOps) > 0 {
		writeDiffs = func(w *tar.Writer) error {
			return ctx.MemFS.AddLayerByCopyOps(ctx.CopyOps, w)
		}
	} else {
		// Nothing to do, return.
		return nil, nil
	}

	gzipTarDigester, tarDigester, tempFileName, err := tarAndGzipDiffs(ctx, writeDiffs)
	if err != nil {
		return nil, fmt.Errorf("failed to generate diff layer: %s", err)
	}
	defer os.Remove(tempFileName)

	tarSHA256 := hex.EncodeToString(tarDigester.Sum(nil))
	gzipTarSHA256 := hex.EncodeToString(gzipTarDigester.Sum(nil))
	if err := ctx.ImageStore.Layers.LinkStoreFileFrom(
		gzipTarSHA256, tempFileName); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("link store file %s from %s: %s", gzipTarSHA256, tempFileName, err)
	}
	info, err := ctx.ImageStore.Layers.GetStoreFileStat(gzipTarSHA256)
	if err != nil {
		return nil, fmt.Errorf("get store file stat %s: %s", gzipTarSHA256, err)
	}

	layerTarDigest := image.Digest("sha256:" + tarSHA256)
	layerGzipDescriptor := image.Descriptor{
		MediaType: image.MediaTypeLayer,
		Size:      info.Size(),
		Digest:    image.Digest("sha256:" + gzipTarSHA256),
	}
	ctx.MustScan = false
	ctx.CopyOps = make([]*snapshot.CopyOperation, 0)
	return []*image.DigestPair{
		{
			TarDigest:      layerTarDigest,
			GzipDescriptor: layerGzipDescriptor,
		},
	}, nil
}
