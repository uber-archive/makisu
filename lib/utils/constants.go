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

// BuildHash is a variable that will be populated at build-time of the
// binary via the ldflags parameter. It is used to break cache when a new
// version of makisu is used.
var BuildHash string

// We need an init function for now to go around the github issue listed above.
func init() {
	if BuildHash == "" {
		BuildHash = "master-unreleased"
	}
}
