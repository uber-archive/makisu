package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ImageStore contains a manifeststore, a layertarstore, and a sandbox dir.
type ImageStore struct {
	RootDir    string
	SandboxDir string
	Manifests  *ManifestStore
	Layers     *LayerTarStore
}

// NewImageStore creates a new ImageStore.
func NewImageStore(rootDir string) (ImageStore, error) {
	sandboxParent := filepath.Join(rootDir, "sandbox")
	if err := os.MkdirAll(sandboxParent, 0755); err != nil {
		return ImageStore{}, fmt.Errorf("init sandbox parent dir: %s", err)
	}
	sandboxDir, err := ioutil.TempDir(sandboxParent, "sandbox")
	if err != nil {
		return ImageStore{}, fmt.Errorf("init sandbox dir: %s", err)
	}

	m, err := NewManifestStore(rootDir)
	if err != nil {
		return ImageStore{}, fmt.Errorf("init manifest store: %s", err)
	}
	l, err := NewLayerTarStore(rootDir)
	if err != nil {
		return ImageStore{}, fmt.Errorf("init layer store: %s", err)
	}

	return ImageStore{
		RootDir:    rootDir,
		SandboxDir: sandboxDir,
		Manifests:  m,
		Layers:     l,
	}, nil
}

// CleanupSandbox removes sandbox dir. This should be done after every build.
func CleanupSandbox(rootDir string) error {
	sandboxParent := filepath.Join(rootDir, "sandbox")
	if err := os.RemoveAll(sandboxParent); err != nil {
		return fmt.Errorf("remove sandbox parent %s: %s", sandboxParent, err)
	}
	return nil
}
