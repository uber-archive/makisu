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

package context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/uber/makisu/lib/archive"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/storage"

	"github.com/andres-erbsen/clock"
)

const (
	_stagesDir = "stages"
)

// BuildContext stores build state for one build stage.
type BuildContext struct {
	RootDir    string // Root of the build file system. Always "/" in production.
	ContextDir string // Source of copy/add operations.

	// MemFS and ImageStore can be shared accross all copies of the BuildContext.
	MemFS      *archive.MemFS     // Merged view of base layers. Layers should be merged in order.
	ImageStore storage.ImageStore // Stores image layers and manifests.

	CopyOps  []*archive.CopyOperation
	MustScan bool

	stagesDir string // Containers dirs with files needed for 'copy --from' operations.
}

// NewBuildContext inits a new BuildContext object.
func NewBuildContext(
	rootDir, contextDir string, imageStore storage.ImageStore) (*BuildContext, error) {

	stagesDir := filepath.Join(imageStore.SandboxDir, _stagesDir)
	if err := os.MkdirAll(stagesDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create stages dir: %s", err)
	}

	blacklist := append(pathutils.DefaultBlacklist, contextDir, imageStore.RootDir)
	memFS, err := archive.NewMemFS(clock.New(), rootDir, blacklist)
	if err != nil {
		return nil, fmt.Errorf("init memfs: %s", err)
	}

	return &BuildContext{
		RootDir:    rootDir,
		ContextDir: contextDir,
		stagesDir:  stagesDir,
		MemFS:      memFS,
		ImageStore: imageStore,
		CopyOps:    make([]*archive.CopyOperation, 0),
		MustScan:   false,
	}, nil
}

// StageDir returns the directory that context from a stage should be written to and read from.
func (ctx *BuildContext) StageDir(alias string) string {
	return filepath.Join(ctx.stagesDir, alias)
}

// Cleanup cleans up files kept across stages after the build is completed.
func (ctx *BuildContext) Cleanup() error {
	return os.RemoveAll(ctx.stagesDir)
}
