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
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/uber/makisu/lib/docker/image"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/storage"
	"github.com/uber/makisu/lib/stream"
)

// ImageTarer contains a Tar function that returns a reader to the resulting
// tar file.
type ImageTarer interface {
	Tar(registry, repo, tag string) (io.Reader, error)
}

// DefaultImageTarer is the default implementation of the ImageTarer interface.
type DefaultImageTarer struct {
	store *storage.ImageStore
}

// NewDefaultImageTarer creates a new DefaultImageTarer with the given
// manifests, layers, and rootdir.
func NewDefaultImageTarer(store *storage.ImageStore) DefaultImageTarer {
	return DefaultImageTarer{
		store: store,
	}
}

// CreateTarReadCloser creates a new tar from the inputs and returns a reader
// that automatically closes on EOF.
func (tarer DefaultImageTarer) CreateTarReadCloser(imageName image.Name) (io.Reader, error) {
	dir, err := tarer.createTarDir(imageName)
	if err != nil {
		return nil, err
	}

	// Create target tar file
	targetPath := dir + ".tar"
	if err := snapshot.CreateTarFromDirectory(targetPath, dir); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	fh, err := os.Open(targetPath)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	reader := stream.NewCloseOnErrorReader(fh, func() error {
		return os.RemoveAll(dir)
	})
	return reader, nil
}

// CreateTarReader creates a new tar from the inputs and returns a simple reader
// to that file.
func (tarer DefaultImageTarer) CreateTarReader(imageName image.Name) (io.Reader, error) {
	dir, err := tarer.createTarDir(imageName)
	if err != nil {
		return nil, err
	}

	// Create target tar file.
	targetPath := dir + ".tar"
	if err := snapshot.CreateTarFromDirectory(targetPath, dir); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return os.Open(targetPath)
}

func (tarer DefaultImageTarer) createTarDir(imageName image.Name) (string, error) {
	// Get the export manifest
	exportManifest, err := tarer.getExportManifest(imageName)
	if err != nil {
		return "", err
	}
	exportManifestData, err := json.Marshal([]image.ExportManifest{exportManifest})
	if err != nil {
		return "", err
	}

	repo, tag := imageName.GetRepository(), imageName.GetTag()
	// Create tmp file for target tar.
	dir := filepath.Join(tarer.store.SandboxDir, repo, tag)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	log.Infof("Image tarrer dir: %s", dir)

	// Write export manifest data to target.
	manifestPath := path.Join(dir, "manifest.json")
	err = ioutil.WriteFile(manifestPath, exportManifestData, perm)
	if err != nil {
		return "", err
	}

	// Link config and layer files to target.
	configPath := path.Join(dir, exportManifest.Config.String())
	err = tarer.store.Layers.LinkStoreFileTo(exportManifest.Config.ID(), configPath)
	if err != nil && !os.IsExist(err) {
		return "", err
	}

	// For each layer, add it to the tarball directory.
	for _, layer := range exportManifest.Layers {
		layerPath := path.Join(dir, layer.String())
		// create layer subdir
		err = os.MkdirAll(path.Dir(layerPath), perm)
		if err != nil {
			return "", err
		}
		err = tarer.store.Layers.LinkStoreFileTo(layer.ID(), layerPath)
		if err != nil && !os.IsExist(err) {
			return "", err
		}
	}
	return dir, nil
}

func (tarer DefaultImageTarer) getExportManifest(imageName image.Name) (image.ExportManifest, error) {
	repo, tag := imageName.GetRepository(), imageName.GetTag()
	manifestReader, err := tarer.store.Manifests.GetStoreFileReader(repo, tag)
	if err != nil {
		return image.ExportManifest{}, err
	}
	manifestData, err := ioutil.ReadAll(manifestReader)
	if err != nil {
		return image.ExportManifest{}, err
	}
	defer manifestReader.Close()

	distribution, _, err := image.UnmarshalDistributionManifest(image.MediaTypeManifest, manifestData)
	if err != nil {
		return image.ExportManifest{}, err
	}
	// create export manifest from distribution manifest.
	exportManifest := image.NewExportManifestFromDistribution(imageName, distribution)
	return exportManifest, nil
}
