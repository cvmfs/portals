package cmd

import (
	"fmt"

	"github.com/cvmfs/portals/lib"
	"github.com/cvmfs/portals/log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(parseConfigCmd)
}

var parseConfigCmd = &cobra.Command{
	Use:     "parse-config",
	Aliases: []string{"parse"},
	Short:   "Parse the config file highlight possible errors",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, arg []string) {

		config, err := lib.ParseConfig(arg[0])
		if err != nil {
			log.LogE(err).Fatal("Error in parsing the configuration file")
		}

		fmt.Printf("%#v\n", config)
	},
}
