package cmd

import (
	//"github.com/siscia/portals/log"

	"github.com/spf13/cobra"
)

var listFilesCmd = &cobra.Command{
	Use:     "list-files",
	Aliases: []string{"ls"},
	Short:   "List the files in the bucket from the config",
	Args:    cobra.MinimumNArgs(1),
	Run:     func(cmd *cobra.Command, arg []string) {},
}
