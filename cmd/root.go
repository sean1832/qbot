/*
Copyright © 2025 qbot <dev@zekezhang.com>
*/
package cmd

import (
	"os"

	util "github.com/sean1832/qbot/pkg/utils"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "qbot",
	Short:   "qbot v" + util.VERSION,
	Long:    `qbittorrent post-processing CLI` + "\n" + "Version: " + util.VERSION,
	Version: util.VERSION,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n") // Only print the version number.
	rootCmd.CompletionOptions.DisableDefaultCmd = true // Disable default command
}
