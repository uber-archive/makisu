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

package fileio

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// ReaderToFile copies the data from a reader to a destination file.
func ReaderToFile(r io.Reader, dst string) error {
	dst = filepath.Clean(dst)
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file %s: %s", dst, err)
	}
	defer w.Close()

	if _, err = io.Copy(w, r); err != nil {
		return fmt.Errorf("copy to file %s: %s", dst, err)
	}
	return nil
}

// ConcatDirectoryContents concatenates all regular files inside the source
// directory and returns their concatenated contents.
func ConcatDirectoryContents(sourceDir string) ([]byte, error) {
	files, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %s", err)
	}

	output := bytes.Buffer{}
	for _, fi := range files {
		path := filepath.Join(sourceDir, fi.Name())
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %s", path, err)
		} else if _, err := output.Write(content); err != nil {
			return nil, fmt.Errorf("write to buffer: %s", err)
		}
	}
	return output.Bytes(), nil
}
