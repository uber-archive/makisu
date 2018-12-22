package cmd

import (
	"fmt"

	"github.com/uber/makisu/lib/utils"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(utils.BuildHash)
	},
}
