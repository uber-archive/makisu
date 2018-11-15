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

package main

import (
	"log"
	"os"

	"github.com/uber/makisu/cli"
	"github.com/uber/makisu/lib/utils"

	"github.com/apourchet/commander"
)

func main() {
	log.Printf("Starting makisu (version=%s)", utils.BuildHash)

	application := cli.NewWorkerApplication()
	cmd := commander.New()
	if err := cmd.RunCLI(application, os.Args[1:]); err != nil {
		log.Fatalf("Command failure: %s", err)
	}
}
