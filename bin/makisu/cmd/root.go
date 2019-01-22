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

package cmd

import (
	"fmt"
	"os"

	"github.com/uber/makisu/lib/log"

	"github.com/spf13/cobra"
)

var (
	LogLevel   string
	LogOutput  string
	LogFormat  string
	CPUProfile bool
)

func getRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "makisu",
		Short: "makisu is a fast Fast and flexible Docker image building tool",
		Long: "makisu is a fast Fast and flexible Docker image building tool " +
			"designed for unprivileged containerized environments like Mesos and Kubernetes. " +
			"More information is available at https://github.com/uber/makisu`.",

		Run: func(ccmd *cobra.Command, args []string) {
			if cleanup, err := processGlobalFlags(); err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				defer cleanup()
			}

			ccmd.HelpFunc()(ccmd, args)
		},
	}

	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "Verbose level of logs. Valid values are \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\"")
	rootCmd.PersistentFlags().StringVar(&LogOutput, "log-output", "stdout", "The output file path for the logs. Set to \"stdout\" to output to stdout")
	rootCmd.PersistentFlags().StringVar(&LogFormat, "log-fmt", "json", "The format of the logs. Valid values are \"json\" and \"console\"")
	rootCmd.PersistentFlags().BoolVar(&CPUProfile, "cpu-profile", false, "Profile the application")

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false
	return rootCmd
}

func Execute() {
	rootCmd := getRootCmd()
	rootCmd.AddCommand(getBuildCmd())
	rootCmd.AddCommand(getVersionCmd())
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
