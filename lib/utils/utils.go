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

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/uber/makisu/lib/log"
)

// MultiErrors contains a list of errors. It supports adding and collecting errors in multiple threads.
type MultiErrors struct {
	sync.Mutex

	errStr string
}

// NewMultiErrors returns a new MultiErrors obj.
func NewMultiErrors() *MultiErrors {
	return &MultiErrors{}
}

// Add appends an error to the list.
func (e *MultiErrors) Add(err error) {
	e.Lock()
	defer e.Unlock()

	if e.errStr == "" {
		e.errStr = err.Error()
		return
	}

	e.errStr = fmt.Sprintf("%s; %s", e.errStr, err.Error())
}

// Collect returns the result error.
func (e *MultiErrors) Collect() error {
	e.Lock()
	defer e.Unlock()

	if e.errStr == "" {
		return nil
	}

	return fmt.Errorf(e.errStr)
}

// Must ensures that the condition passed in is true.
// If condition is true this function NO-OPS; Otherwise it logs the message
// formatted with the arguments passed in
func Must(condition bool, message string, arguments ...interface{}) {
	if !condition {
		log.Fatalf(message, arguments...)
	}
}

// DefaultEnv returns the environment variable <key> if it is found,
// and _default otherwise.
func DefaultEnv(key string, _default string) string {
	val, found := os.LookupEnv(key)
	if !found {
		return _default
	}
	return val
}

// ConvertStringSliceToMap parses a string slice as "=" separated key value
// pairs, and returns a map.
func ConvertStringSliceToMap(values []string) map[string]string {
	result := make(map[string]string)
	for _, value := range values {
		pair := strings.SplitN(value, "=", 2)
		if len(pair) == 1 {
			result[pair[0]] = ""
		} else {
			result[pair[0]] = pair[1]
		}
	}

	return result
}

// MergeEnv merges a new env key value pair into existing list.
// This is needed because Docker image config defines Env as []string, but
// actually uses it as map[string]string.
func MergeEnv(envList []string, newEnvMap map[string]string) []string {
	envMap := ConvertStringSliceToMap(envList)
	for newK, newV := range newEnvMap {
		envMap[newK] = newV
	}

	result := []string{}
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(result)
	return result
}

// MergeStringMaps merges two string maps and returns the combined map.
// If there are duplicate keys it picks value from second map.
func MergeStringMaps(mapOne, mapTwo map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range mapOne {
		result[k] = v
	}
	for k, v := range mapTwo {
		result[k] = v
	}
	return result
}

// MergeStructMaps merges two struct maps and returns the combined map.
// If there are duplicate keys it picks value from second map.
func MergeStructMaps(mapOne, mapTwo map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for k, v := range mapOne {
		result[k] = v
	}
	for k, v := range mapTwo {
		result[k] = v
	}
	return result
}

// Min returns the minimum value of the integers passed in as arguments.
func Min(a int64, others ...int64) int64 {
	min := a
	for _, other := range others {
		if other < min {
			min = other
		}
	}
	return min
}

// IsSpecialFile returns true for file types that overlayfs ignores.
// Overlayfs logic:
//   #define special_file(m) (S_ISCHR(m)||S_ISBLK(m)||S_ISFIFO(m)||S_ISSOCK(m))
func IsSpecialFile(fi os.FileInfo) bool {
	return fi.Mode()&(os.ModeCharDevice|os.ModeDevice|os.ModeNamedPipe|os.ModeSocket) != 0
}

// FileInfoStat provides a convenience wrapper for casting the generic
// FileInfo.Sys field.
func FileInfoStat(fi os.FileInfo) *syscall.Stat_t {
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		panic("Failed to cast fileinfo.sys to stat_t")
	}
	return s
}

// GetUIDGID returns the uid/gid pair for the current user.
func GetUIDGID() (int, int, error) {
	return os.Geteuid(), os.Getegid(), nil
}

// IsValidJSON returns true if the blob passed in is a valid json object.
func IsValidJSON(blob []byte) bool {
	into := map[string]interface{}{}
	return json.Unmarshal(blob, &into) == nil
}
