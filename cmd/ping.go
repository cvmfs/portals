package cmd

import (
	"sync"

	"github.com/siscia/portals/lib"
	"github.com/siscia/portals/log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pingCmd)
}

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Start a PING subprocess against the status buckets",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, arg []string) {
		config, err := lib.ParseConfig(arg[0])
		if err != nil {
			log.LogE(err).Error("Error in parsing the configuration file")
			return
		}
		var wg sync.WaitGroup
		for _, bucketConfiguration := range config.Credentials {

			couple, err := lib.NewS3BucketCouple(bucketConfiguration)
			if err != nil {
				log.LogE(err).Error("Error in generating the Couple of Buckets")
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				lib.UploadPingToStatusBucket(couple)
			}()
		}
		wg.Wait()
	},
}
