package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "portals",
	Short: "Manage the portals daemon, show available command.",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func EntryPoint() {
	rootCmd.Execute()
}
