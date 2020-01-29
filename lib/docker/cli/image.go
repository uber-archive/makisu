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
	"fmt"
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


// DefaultImageTarer exports/imports images from an ImageStore
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

// CreateTarReadCloser exports an image from the image store as a tar, and returns a reader for the tar
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

// CreateTarReader exports an image from the image store as a tar, and returns a reader for the tar
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

// WriteTar imports an image, as a tar, to the image store
func (tarer DefaultImageTarer) WriteTar(imageName image.Name, tarPath string) error {
	repo, tag := imageName.GetRepository(), imageName.GetTag()

	// Extract tar into temporary directory
	dir := filepath.Join(tarer.store.SandboxDir, repo, tag)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create unpack directory: %s", err)
	}
	defer os.RemoveAll(dir)

	if err := snapshot.CreateDirectoryFromTar(dir, tarPath); err != nil {  // TODO
		return fmt.Errorf("unpack tar: %s", err)
	}

	// Read manifest
	exportManifestPath := filepath.Join(dir, "manifest.json")
	var exportManifests []image.ExportManifest

	if exportManifestJSON, err := ioutil.ReadFile(exportManifestPath); err != nil {
		return fmt.Errorf("read manifest: %s", err)
	} else if err := json.Unmarshal(exportManifestJSON, exportManifests); err != nil {
		return fmt.Errorf("unmarshal manifest: %s", err)
	}

	for _, exportManifest := range exportManifests {

		// Import extracted dir content into image store -- manifest.json
		distManifest, err := image.NewDistributionManifestFromExport(exportManifest, dir)
		if err != nil {
			return fmt.Errorf("create distribution manifest: %s", err)
		}
		distManifestJSON, err := json.Marshal(distManifest)
		if err != nil {
			return fmt.Errorf("marshal manifest to JSON: %s", err)
		}

		distManifestFile, err := ioutil.TempFile(tarer.store.SandboxDir, "")
		if err != nil {
			return fmt.Errorf("create tmp manifest file: %s", err)
		}
		if _, err := distManifestFile.Write(distManifestJSON); err != nil {
			return fmt.Errorf("write manifest file: %s", err)
		}
		if err := distManifestFile.Close(); err != nil {
			return fmt.Errorf("close manifest file: %s", err)
		}

		distManifestPath := distManifestFile.Name()
		if err = tarer.store.Manifests.LinkStoreFileFrom(repo, tag, distManifestPath); err != nil && !os.IsExist(err) {
			return fmt.Errorf("commit manifest to store: %s", err)
		}

		// Import extracted dir content into image store -- {sha}.json
		configPath := filepath.Join(dir, exportManifest.Config.String())
		configID := exportManifest.Config.ID()
		if err = tarer.store.Layers.LinkStoreFileFrom(configID, configPath); err != nil && !os.IsExist(err) {
			return fmt.Errorf("commit config to store: %s", err)
		}

		// Import extracted dir content into image store -- {sha}/layer.tar
		for _, layer := range exportManifest.Layers {
			layerPath := path.Join(dir, layer.String())
			layerID := layer.ID()
			if err = tarer.store.Layers.LinkStoreFileFrom(layerID, layerPath); err != nil && !os.IsExist(err) {
				return fmt.Errorf("commit layer to store: %s", err)
			}
		}
	}

	return nil
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
