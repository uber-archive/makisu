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
	"runtime/pprof"

	"github.com/uber/makisu/lib/log"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.PersistentFlags().StringVar(&LogLevel, "log-level", "info", "Verbose level of logs. Valid values are \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\"")
	rootCmd.PersistentFlags().StringVar(&LogOutput, "log-output", "stdout", "The output file path for the logs. Set to \"stdout\" to output to stdout")
	rootCmd.PersistentFlags().StringVar(&LogFormat, "log-fmt", "json", "The format of the logs. Valid values are \"json\" and \"console\"")
	rootCmd.PersistentFlags().BoolVar(&CpuProfile, "cpu-profile", false, "Profile the application")

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false
}

var (
	LogLevel   string
	LogOutput  string
	LogFormat  string
	CpuProfile bool

	rootCmd = &cobra.Command{
		Use:   "makisu",
		Short: "makisu is a fast Fast and flexible Docker image building tool",
		Long: "makisu is a fast Fast and flexible Docker image building tool " +
			"designed for unprivileged containerized environments like Mesos and Kubernetes. " +
			"More information is available at https://github.com/uber/makisu`.",

		Run: func(ccmd *cobra.Command, args []string) {
			ccmd.HelpFunc()(ccmd, args)
		},
	}
)

func Execute() {
	// Initializes logger.
	logger, err := getLogger()
	if err != nil {
		fmt.Println("Configure logger: ", err)
		os.Exit(1)
	}
	log.SetLogger(logger.Sugar())

	if CpuProfile {
		// Set up profiling.
		if err := setupProfiler(); err != nil {
			fmt.Println("Setup profiler: ", err)
			os.Exit(1)
		}

		defer pprof.StopCPUProfile()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	if LogOutput != "stdout" {
		config.OutputPaths = []string{LogOutput}
	}

	if err := config.Level.UnmarshalText([]byte(LogLevel)); err != nil {
		return nil, fmt.Errorf("parse log level: %s", err)
	}

	config.Encoding = LogFormat
	config.DisableStacktrace = true
	config.DisableCaller = true
	return config.Build()
}

func setupProfiler() error {
	f, err := os.Create("/tmp/makisu.prof")
	if err != nil {
		return fmt.Errorf("create profile file: %s", err)
	}
	pprof.StartCPUProfile(f)

	return nil
}
