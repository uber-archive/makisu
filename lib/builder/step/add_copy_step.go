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
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uber/makisu/lib/context"
	"github.com/uber/makisu/lib/pathutils"
	"github.com/uber/makisu/lib/snapshot"
	"github.com/uber/makisu/lib/utils"
)

// addCopyStep implements BuildStep and execute ADD/COPY directive
// From docker official documentation, COPY obeys the following rules:
// - The <src> path must be inside the context of the build; you cannot COPY ../something /something, because the first
//   step of a docker build is to send the context directory (and subdirectories) to the docker daemon.
// - If <src> is a directory, the entire contents of the directory are copied, including filesystem metadata.
//   Note: The directory itself is not copied, just its contents.
// - If <src> is any other kind of file, it is copied individually along with its metadata. In this case, if <dest> ends
//   with a trailing slash /, it will be considered a directory and the contents of <src> will be written at
//   <dest>/base(<src>).
// - If multiple <src> resources are specified, either directly or due to the use of a wildcard, then <dest> must be a
//   directory, and it must end with a slash /.
// - If <dest> does not end with a trailing slash, it will be considered a regular file and the contents of <src> will
//   be written at <dest>.
// - If <dest> doesn’t exist, it is created along with all missing directories in its path.
// So in summarry there are 6 supported cases:
// - COPY file1 /target/file1
// - COPY file1 /target/dir1/
// - COPY file1, file2 ... /tmp/dir1/
// - COPY dir1  /target/dir1/
// - COPY dir1  /target/dir1  (same as prev)
// - COPY dir1, dir2 ...   /tmp/dir1/
// It also supports a "from" flag to specify a prev stage to copy files from.
type addCopyStep struct {
	*baseStep

	fromStage string
	fromPaths []string
	toPath    string
	chown     string
}

// newAddCopyStep returns a BuildStep from given arguments.
func newAddCopyStep(
	directive Directive, args, chown, fromStage string,
	fromPaths []string, toPath string, commit bool) (*addCopyStep, error) {

	if len(fromPaths) > 1 && !(strings.HasSuffix(toPath, "/") || toPath == "." || toPath == "..") {
		return nil, fmt.Errorf("copying multiple source files, target must be a directory ending in \"/\"")
	}
	return &addCopyStep{
		baseStep:  newBaseStep(directive, args, commit),
		fromStage: fromStage,
		fromPaths: fromPaths,
		toPath:    toPath,
		chown:     chown,
	}, nil
}

// RequireOnDisk returns true if the add/copy has a chown argument, as we need
// to read the users file to translate user/group name to uid/gid.
func (s *addCopyStep) RequireOnDisk() bool { return s.chown != "" }

// ContextDirs returns the stage and directories that a 'COPY --from=<stage>' depends on.
func (s *addCopyStep) ContextDirs() (string, []string) {
	if s.fromStage == "" {
		return "", nil
	}
	return s.fromStage, s.fromPaths
}

// SetCacheID sets the cache ID of the step given a seed SHA256 value. Calculates the
// ID randomly if copying from another stage, else checksums the file contents.
func (s *addCopyStep) SetCacheID(ctx *context.BuildContext, seed string) error {
	if s.fromStage != "" {
		// It is copying from a previous stage, return random bytes.
		// TODO: support cache here by chaining stage sha.
		b := make([]byte, 4)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if _, err := r.Read(b); err != nil {
			return fmt.Errorf("read rand: %s", err)
		}
		s.cacheID = fmt.Sprintf("%x", b)
	} else {
		// Initialize the checksum with the seed, directive and args.
		checksum := crc32.NewIEEE()
		_, err := checksum.Write([]byte(seed + string(s.directive) + s.args))
		if err != nil {
			return fmt.Errorf("hash copy directive: %s", err)
		}

		// If the step args and the contents of sources are identical,
		// we should be able to use the cache from the previous build.
		if err := s.updateContextChecksum(ctx, checksum); err != nil {
			return fmt.Errorf("hash context sources: %s", err)
		}
		s.cacheID = fmt.Sprintf("%x", checksum.Sum32())
	}
	return nil
}

// Execute executes the add/copy step. If modifyFS is true, actually performs the on-disk copy.
func (s *addCopyStep) Execute(ctx *context.BuildContext, modifyFS bool) (err error) {
	sourceRoot := s.contextRootDir(ctx)
	sources := s.resolveFromPaths(ctx)
	relPaths := make([]string, len(sources))
	for i, source := range sources {
		relPaths[i], err = pathutils.TrimRoot(source, sourceRoot)
		if err != nil {
			return fmt.Errorf("trim root: %s", err)
		}
	}

	internal := s.fromStage != ""
	blacklist := append(pathutils.DefaultBlacklist, ctx.ImageStore.RootDir)
	copyOp, err := snapshot.NewCopyOperation(
		relPaths, sourceRoot, s.workingDir, s.toPath, s.chown, blacklist, internal)
	if err != nil {
		return fmt.Errorf("invalid copy operation: %s", err)
	}

	ctx.CopyOps = append(ctx.CopyOps, copyOp)
	if modifyFS {
		return copyOp.Execute()
	}
	return nil
}

// Updates the checksum passed in with the data stored in the context on the filesystem.
func (s *addCopyStep) updateContextChecksum(ctx *context.BuildContext, checksum io.Writer) error {
	if s.fromStage != "" {
		return fmt.Errorf("not supported: the copy step has from stage flag")
	}

	for _, source := range s.resolveFromPaths(ctx) {
		if err := filepath.Walk(source, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("prev error during walk: %s", err)
			}
			return checksumPathContents(path, fi, checksum)
		}); err != nil {
			return fmt.Errorf("walk %s: %s", source, err)
		}
	}
	return nil
}

func (s *addCopyStep) resolveFromPaths(ctx *context.BuildContext) []string {
	root := s.contextRootDir(ctx)
	sources := []string{}
	for _, source := range s.fromPaths {
		source = filepath.Join(root, source)
		matches, err := filepath.Glob(source)
		if err != nil || len(matches) == 0 {
			sources = append(sources, source)
		} else {
			sources = append(sources, matches...)
		}
	}
	return sources
}

func (s *addCopyStep) contextRootDir(ctx *context.BuildContext) string {
	if s.fromStage != "" {
		return ctx.CopyFromRoot(s.fromStage)
	}
	return ctx.ContextDir
}

func checksumPathContents(path string, fi os.FileInfo, checksum io.Writer) error {
	// Skip special files.
	if utils.IsSpecialFile(fi) {
		if fi.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}

	if _, err := checksum.Write([]byte(path)); err != nil {
		return fmt.Errorf("write path to checksum")
	}

	// If it is a directory, just return after checksumming the dir name.
	if fi.IsDir() {
		return nil
	}

	// If it's a symlink, don't follow.
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return fmt.Errorf("read link %s: %s", path, err)
		}
		_, err = checksum.Write([]byte(target))
		return err
	}

	fh, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %s", path, err)
	}
	if _, err := io.Copy(checksum, fh); err != nil {
		return fmt.Errorf("read %s: %s", path, err)
	}
	return nil
}
