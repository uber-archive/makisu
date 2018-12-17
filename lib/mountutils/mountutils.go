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

package mountutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/uber/makisu/lib/log"
)

type mountInfo struct {
	// Map from filename to mount point description.
	data       map[string]mountPoint
	init       sync.Once
	mountsFile string
}

// mountPoint defines a single row of the /proc/mounts file.
type mountPoint struct {
	Source  string
	Target  string
	FSType  string
	Options string
}

var defaultInfo = &mountInfo{
	data:       map[string]mountPoint{},
	mountsFile: "/proc/mounts",
}

func newMountInfo() *mountInfo {
	return &mountInfo{
		data:       map[string]mountPoint{},
		mountsFile: "/proc/mounts",
	}
}

func (info *mountInfo) initialize() error {
	content, err := ioutil.ReadFile(info.mountsFile)
	if os.IsNotExist(err) {
		log.Debug("Cannot init mountmanager, /proc/mounts does not exist")
		return nil
	} else if err != nil {
		return fmt.Errorf("mountmanager initialize: %s", err)
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		fields := strings.SplitN(line, " ", 4)
		if len(fields) < 4 {
			return fmt.Errorf("cannot parse mounts file %s", info.mountsFile)
		} else if fields[1] == "/" {
			// Skip this entry which usually appears with overlayfs.
			continue
		}
		info.data[fields[1]] = mountPoint{
			Source:  fields[0],
			Target:  fields[1],
			FSType:  fields[2],
			Options: fields[3],
		}
	}
	return nil
}

func (info *mountInfo) isMountpoint(filename string) (bool, error) {
	var err error
	info.init.Do(func() { err = info.initialize() })
	if err != nil {
		return false, fmt.Errorf("ismount: %s", err)
	}
	_, found := info.data[filename]
	return found, nil
}

func (info *mountInfo) isMounted(filename string) (bool, error) {
	if mp, err := info.isMountpoint(filename); err != nil || mp {
		return mp, err
	}
	for path := range info.data {
		if path[len(path)-1] != '/' {
			path += "/"
		}
		if strings.HasPrefix(filename, path) {
			return true, nil
		}
	}
	return false, nil
}

func (info *mountInfo) containsMountpoint(filename string) (bool, error) {
	if mp, err := info.isMountpoint(filename); err != nil || mp {
		return mp, err
	}
	for path := range info.data {
		if path[len(path)-1] != '/' {
			path += "/"
		}
		if filepath.HasPrefix(path, filename) {
			return true, nil
		}
	}
	return false, nil
}

// IsMountpoint returns true if the file is a mountpoint, with an error if
// there was a problem reading the mountpoint information. Returns false
// on every file with no error if the mounts file was not found.
func IsMountpoint(filename string) (bool, error) {
	return defaultInfo.isMountpoint(filename)
}

// IsMounted returns true if the file is located inside a mounted directory
// of the current filesystem, and false otherwise. Returns false on every
// file with no error if the mounts file was not found.
func IsMounted(filename string) (bool, error) {
	return defaultInfo.isMounted(filename)
}

// ContainsMountpoint returns true if the file is a mountpoint, or if the
// file is a directory that contains a mountpoint somewhere in it.
func ContainsMountpoint(filename string) (bool, error) {
	return defaultInfo.containsMountpoint(filename)
}
