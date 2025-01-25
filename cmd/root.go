package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for your CLI. It doesn’t run anything itself
// unless the user runs it with no subcommands.
var rootCmd = &cobra.Command{
	Use:   "brio",
	Short: "Brio is a CLI tool for extracting annotated code snippets.",
	Long: `Brio is a command-line utility that scans your codebase
for snippet annotations in comments and extracts only the relevant sections 
you've tagged, enabling a streamlined workflow for LLM-based code analysis.`,
	// Run: if you want the root command to do something by default
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("Please use a subcommand (e.g., 'extract').")
	// },
}

// Execute is called by main.go to run the root command.
// If an error occurs, we print to stderr and exit.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init runs before main() and sets up persistent flags or subcommands.
func init() {
	// Here, you can set up global persistent flags if you like, for example:
	// rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")

	// We add the subcommands here, or you can do so in their init() functions.
	// In this example, the extract subcommand is added in extract.go’s init().
}
