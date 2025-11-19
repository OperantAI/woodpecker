package cmd

import (
	"github.com/operantai/woodpecker/cmd/woodpecker-postman-collection/postman"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "postman-collection",
	Short: "Run Postman collections as Woodpecker components",
	Long:  "Run Postman collections as Woodpecker components",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		output.WriteError("%s", err.Error())
	}
}

// cleanCmd represents the clean command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a Postman collection as a Woodpecker component",
	Long:  "Run a Postman collection as a Woodpecker component",
	Run: func(cmd *cobra.Command, args []string) {
		output.WriteInfo("Postman component management is on the process!")
		collectionPath, error := cmd.Flags().GetString("collection")
		if error != nil {
			output.WriteError("Error reading collection flag: %v", error)
		}
		postman.RunCollection(collectionPath)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("collection", "c", "", "Path to the Postman collection JSON file")
	_ = runCmd.MarkFlagRequired("collection")
}
