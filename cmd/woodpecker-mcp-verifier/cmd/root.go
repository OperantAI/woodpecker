// Package cmd contains the cli commands to start and run the MCP client verifier
package cmd

import (
	"context"
	"strings"

	"github.com/operantai/woodpecker/cmd/woodpecker-mcp-verifier/utils"
	mcpverifier "github.com/operantai/woodpecker/internal/mcp-verifier"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "mcp-verifier",
		Short: "Run a MCP client verifier as a Woodpecker components",
		Long:  "Run a MCP client verifier as a Woodpecker components",
	}
	protocol utils.MCMCPprotocol
	cmdArgs  []string
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
		output.WriteInfo("MCP client verifier starting ...")
		var serverURL, payloadPath string
		var err error

		if serverURL, err = cmd.Flags().GetString("url"); err != nil {
			output.WriteFatal("%v", err)
		}
		payloadPath = viper.GetString("payload-path")

		if err := mcpverifier.RunClient(context.Background(), serverURL, protocol, &cmdArgs, payloadPath); err != nil {
			output.WriteFatal("%v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	// Tells Viper to use this prefix when reading environment variables
	viper.SetEnvPrefix("woodpecker")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	runCmd.Flags().StringP("url", "u", "", "The MCP server url")
	runCmd.Flags().VarP(&protocol, "protocol", "p", "The MCP protocol being used")
	runCmd.Flags().StringP("payload-path", "t", "/app/payload.json", "The path to the json payload content")
	runCmd.Flags().StringSliceP("cmd_args", "c", cmdArgs, `If STDIO protocol, a comma separated list of cmd and args. i.e -c "uv,run,server"`)
	if err := runCmd.MarkFlagRequired("url"); err != nil {
		output.WriteFatal("%v", err)
	}
	if err := runCmd.MarkFlagRequired("protocol"); err != nil {
		output.WriteFatal("%v", err)
	}
	if err := viper.BindPFlag("payload-path", runCmd.Flags().Lookup("payload-path")); err != nil {
		output.WriteFatal("%v", err)
	}

	// Sets App name
	appName := viper.GetString("APP_NAME")
	if appName == "" {
		output.WriteInfo("Setting WOODPECKER_APP_NAME to woodpecker-mcp-verifier")
		viper.Set("APP_NAME", "woodpecker-mcp-verifier")
	}
}
