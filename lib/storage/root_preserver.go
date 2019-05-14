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

package storage

import (
	"fmt"
	"os"
	"path"

	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/log"
)

const rootPreserverBackupDir = "initial_root"

// RootPreserver contains the locations of:
//  - the "old" root, the one before running makisu that will get removed by `MemFS.Remove()` at the end of the build command
//  - the "saved" root, a dir containing a copy of everyfile of the "old" root
type RootPreserver struct {
	InitialRootDir string
	SavedRootDir   string
	blacklist      []string
}

// NewRootPreserver creates a new RootPreserver.
func NewRootPreserver(rootDir, storageDir string, blacklist []string) (*RootPreserver, error) {
	backupDir := path.Join(storageDir, rootPreserverBackupDir)

	if err := copyOldRootToBackup(rootDir, backupDir, blacklist); err != nil {
		return nil, fmt.Errorf("root preserver failed copying: %s", err)
	}

	return &RootPreserver{
		InitialRootDir: rootDir,
		SavedRootDir:   backupDir,
	}, nil
}

func copyOldRootToBackup(rootDir, backupDir string, blacklist []string) error {
	copier := fileio.NewCopier(blacklist)

	// Remove and recreate backup dir.
	os.RemoveAll(backupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("Failed to create layer download dir %s: %s", backupDir, err)
	}

	// TODO: Handle uid, gid preservation
	if err := copier.CopyDirPreserveOwner(rootDir, backupDir); err != nil {
		return fmt.Errorf("copy dir %s to dir %s: %s", rootDir, backupDir, err)
	}

	return nil
}

// RestoreRoot will copy the backupDir in the initial root dir
func (r *RootPreserver) RestoreRoot() error {
	copier := fileio.NewCopier(r.blacklist)

	// TODO: Handle uid, gid preservation
	if err := copier.CopyDirPreserveOwner(r.SavedRootDir, r.InitialRootDir); err != nil {
		return fmt.Errorf("copy dir %s to dir %s: %s", r.SavedRootDir, r.InitialRootDir, err)
	}

	if err := os.RemoveAll(r.SavedRootDir); err != nil {
		return fmt.Errorf("remove saved root dir %s: %s", r.SavedRootDir, err)
	}

	return nil
}
