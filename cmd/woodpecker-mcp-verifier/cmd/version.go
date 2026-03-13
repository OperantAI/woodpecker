package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Build info which gets populated by ldflags at build time
	Version   string
	GitCommit string
	BuildDate string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Output CLI version information",
	Long:  "Output CLI version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("version %s, build %s, built on %s\n", Version, GitCommit, BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
