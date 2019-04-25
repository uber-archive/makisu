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

package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/utils"
)

// CopyOperation defines a copy operation that occurred to generate a layer from.
type CopyOperation struct {
	srcRoot string
	srcs    []string
	dst     string
	uid     int
	gid     int

	blacklist []string
	// Indicates if the copy op is used for copying from previous stages.
	internal bool
}

// NewCopyOperation initializes and validates a CopyOperation. Use "internal" to
// specify if the copy op is used for copying from previous stages.
func NewCopyOperation(
	srcs []string, srcRoot, workDir, dst, chown string,
	blacklist []string, internal bool) (*CopyOperation, error) {

	if err := checkCopyParams(srcs, workDir, dst); err != nil {
		return nil, fmt.Errorf("check copy param: %s", err)
	}

	uid, gid, err := utils.ResolveChown(chown)
	if err != nil {
		return nil, fmt.Errorf("resolve chown: %s", err)
	}

	relSources := make([]string, len(srcs))
	for k, src := range srcs {
		relSources[k] = pathutils.RelPath(src)
	}

	dst = resolveDestination(workDir, dst)

	return &CopyOperation{
		srcRoot:   srcRoot,
		srcs:      relSources,
		dst:       dst,
		uid:       uid,
		gid:       gid,
		blacklist: blacklist,
		internal:  internal,
	}, nil
}

// Execute performs the actual copying of files specified by the CopyOperation.
func (c *CopyOperation) Execute() error {
	var err error
	for _, src := range c.srcs {
		src, err = evalSymlinks(src, c.srcRoot)
		if err != nil {
			return fmt.Errorf("eval symlinks for %s: %s", src, err)
		}
		src = filepath.Join(c.srcRoot, src)
		fi, err := os.Lstat(src)
		if err != nil {
			return fmt.Errorf("lstat %s: %s", src, err)
		}
		var copier fileio.Copier
		if c.internal {
			copier = fileio.NewInternalCopier()
		} else {
			copier = fileio.NewCopier(c.blacklist)
		}
		if fi.IsDir() {
			// Dir to dir
			if err := copier.CopyDir(src, c.dst, c.uid, c.gid); err != nil {
				return fmt.Errorf("copy dir %s to dir %s: %s", src, c.dst, err)
			}
		} else if isDirFormat(c.dst) {
			// File to dir
			targetFilePath := filepath.Join(c.dst, filepath.Base(src))
			if err := copier.CopyFile(src, targetFilePath, c.uid, c.gid); err != nil {
				return fmt.Errorf("copy file %s to dir %s: %s", src, targetFilePath, err)
			}
		} else {
			// File to file
			if err := copier.CopyFile(src, c.dst, c.uid, c.gid); err != nil {
				return fmt.Errorf("copy file %s to file %s: %s", src, c.dst, err)
			}
		}
	}
	return nil
}

func resolveDestination(workDir, dst string) string {
	if filepath.IsAbs(dst) {
		return dst
	}
	absDst := filepath.Join(workDir, dst)
	// Preserve trailing "/".
	if isDirFormat(dst) {
		absDst += "/"
	}
	return absDst
}

func checkCopyParams(srcs []string, workDir, dst string) error {
	if len(srcs) == 0 {
		return fmt.Errorf("srcs cannot be empty")
	} else if len(srcs) > 1 && !isDirFormat(dst) {
		return fmt.Errorf("tarring multiple sources, destination must end with \"/\"")
	} else if !filepath.IsAbs(dst) && !filepath.IsAbs(workDir) {
		return fmt.Errorf("dst is not absolute path, must specify absolute working directory")
	}
	return nil
}

func isDirFormat(dst string) bool {
	return strings.HasSuffix(dst, "/") || dst == "." || dst == ".."
}
