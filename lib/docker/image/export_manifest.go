package image

import (
	"fmt"
	"path"
	"strings"
)

// Name of files after untar a docker image
const (
	ExportManifestFileName    = "manifest.json"
	layerTarFileName          = "layer.tar"
	legacyImageConfigFileName = "json"
)

// ExportManifest is used for docker load and docker save.
// It contains a list of layer IDs, image config ID, and <repo>:<tag>
type ExportManifest struct {
	Config   ExportConfig
	RepoTags []string
	Layers   []ExportLayer
}

// ExportLayer is a string in the format <layerID>/layer.tar
type ExportLayer string

// ID returns layer ID
func (l ExportLayer) ID() string {
	return strings.TrimSuffix(string(l), "/layer.tar")
}

func (l ExportLayer) String() string {
	return string(l)
}

// ExportConfig is a string in the format <configID>.json
type ExportConfig string

// ID returns config ID
func (c ExportConfig) ID() string {
	return strings.TrimSuffix(string(c), ".json")
}

func (c ExportConfig) String() string {
	return string(c)
}

// NewExportManifestFromDistribution creates ExportManifest given repo, tag and distrubtion manifest
func NewExportManifestFromDistribution(imageName Name, distribution DistributionManifest) ExportManifest {
	exportConfig := ExportConfig(fmt.Sprintf("%s.%s", distribution.Config.Digest.Hex(), legacyImageConfigFileName))
	var exportLayers []ExportLayer
	for _, layer := range distribution.Layers {
		exportLayer := ExportLayer(path.Join(layer.Digest.Hex(), layerTarFileName))
		exportLayers = append(exportLayers, exportLayer)
	}

	exportRepoTags := []string{imageName.String()}

	return ExportManifest{
		Config:   exportConfig,
		RepoTags: exportRepoTags,
		Layers:   exportLayers,
	}
}
