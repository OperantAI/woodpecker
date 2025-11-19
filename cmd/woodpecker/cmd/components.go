/*
Copyright 2023 Operant AI
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/operantai/woodpecker/internal/components"
	"github.com/operantai/woodpecker/internal/output"
	"github.com/operantai/woodpecker/internal/snippets"
	"github.com/spf13/cobra"
)

// cleanCmd represents the clean command
var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Manage woodpecker optional components",
	Long:  "Manage woodpecker optional components",
	Run: func(cmd *cobra.Command, args []string) {
		allComponents := components.ListComponents()
		table := output.NewTable([]string{"Type", "Description"})
		for componentType, description := range allComponents {
			table.AddRow([]string{componentType, description})
		}
		table.Render()
	},
}

// installComponentCmd installs a given component YAML
var installComponentCmd = &cobra.Command{
	Use:   "install",
	Short: "Install a component",
	Run: func(cmd *cobra.Command, args []string) {
		files, err := cmd.Flags().GetStringSlice("files")
		if err != nil {
			output.WriteError("Error reading file flag: %v", err)
		}

		ctx := cmd.Context()
		comp := components.New(ctx)
		if err := comp.Add(files); err != nil {
			output.WriteError("Error installing components %s: %v", strings.Join(files, ","), err)
		}
	},
}

// snippetComponentCmd outputs a template of a given component
var snippetComponentCmd = &cobra.Command{
	Use:   "snippet",
	Short: "Print a template of a component out to stdout",
	Run: func(cmd *cobra.Command, args []string) {
		component, err := cmd.Flags().GetString("component")
		if err != nil {
			output.WriteError("Error reading component flag: %v", err)
		}
		snippet, err := snippets.GetComponentTemplate(component)
		if err != nil {
			output.WriteFatal("Error retrieving component template: %v", err)
		}
		fmt.Println(string(snippet))
	},
}

var uninstallComponentCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall a component",
	Run: func(cmd *cobra.Command, args []string) {
		files, err := cmd.Flags().GetStringSlice("files")
		if err != nil {
			output.WriteError("Error reading file flag: %v", err)
		}

		ctx := cmd.Context()
		comp := components.New(ctx)
		if err := comp.Remove(files); err != nil {
			output.WriteError("Error uninstalling components %s: %v", strings.Join(files, ","), err)
		}
	},
}

func init() {
	rootCmd.AddCommand(componentCmd)
	componentCmd.AddCommand(installComponentCmd)
	componentCmd.AddCommand(uninstallComponentCmd)
	componentCmd.AddCommand(snippetComponentCmd)

	// Define the path of the experiment file to run
	installComponentCmd.Flags().StringSliceP("files", "f", []string{}, "Component files to install")
	_ = installComponentCmd.MarkFlagRequired("files")

	uninstallComponentCmd.Flags().StringSliceP("files", "f", []string{}, "Component files to install")
	_ = uninstallComponentCmd.MarkFlagRequired("files")

	snippetComponentCmd.Flags().StringP("component", "c", "", "Component to generate a template of")
	_ = snippetComponentCmd.MarkFlagRequired("component")
}
