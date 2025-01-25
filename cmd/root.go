package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "brio",
	Short: "Brio is a CLI tool for extracting annotated code snippets.",
	Long: `Brio scans your codebase for tags in comments like:
# start: {"foundation": ["messages"]}
... code ...
# end: {"foundation": ["messages"]}
and extracts relevant snippets based on specified categories.`,
}

// Execute is called by main.go to run the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}

func init() {
	// Here, we can register global/persistent flags if needed.
	// For example:
	// rootCmd.PersistentFlags().BoolVar(&someGlobalFlag, "global", false, "A global flag")
}
