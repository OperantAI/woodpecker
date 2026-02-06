package cmd

import (
	"context"

	mcpverifier "github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/mcp-verifier"
	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var (

	rootCmd = &cobra.Command{
		Use:   "mcp-verifier",
		Short: "Run a MCP client verifier as a Woodpecker components",
		Long:  "Run a MCP client verifier as a Woodpecker components",
	}
	protocol utils.MCMCPprotocol
	cmdArgs []string
)
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
	Short: "Run a MCP client verifier as a Woodpecker component",
	Long:  "Run a MCP client verifier as a Woodpecker component",
	Run: func(cmd *cobra.Command, args []string) {
		output.WriteInfo("MCP client verifying starting ...")
		serverUrl, error := cmd.Flags().GetString("url")
		if error != nil {
			output.WriteError("Error reading collection flag: %v", error)
		}

		if err := mcpverifier.RunClient(context.Background(), serverUrl, protocol, &cmdArgs); err != nil { panic(err) }
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("url", "u", "", "The MCP server url")
	runCmd.Flags().VarP(&protocol, "protocol", "p", "The MCP protocol being used")
	runCmd.Flags().StringSliceP("cmd_args", "c", cmdArgs, `If STDIO protocol, a comma separated list of cmd and args. i.e -c "uv,run,server"`)
	if err := runCmd.MarkFlagRequired("url"); err != nil { panic(err) }
	if err := runCmd.MarkFlagRequired("protocol"); err != nil { panic(err) }
}
