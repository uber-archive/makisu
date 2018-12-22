package cmd

import (
	"fmt"

	"github.com/uber/makisu/lib/utils"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(helpCmd)
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Display usage information for Makisu",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(utils.BuildHash)
	},
}
