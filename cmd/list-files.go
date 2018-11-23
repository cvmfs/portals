package cmd

import (
	"fmt"

	"github.com/cvmfs/portals/lib"
	"github.com/cvmfs/portals/log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listFilesCmd)
}

var listFilesCmd = &cobra.Command{
	Use:     "list-files",
	Aliases: []string{"ls"},
	Short:   "List the files in the bucket from the config",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, arg []string) {
		config, err := lib.ParseConfig(arg[0])
		if err != nil {
			log.LogE(err).Error("Error in parsing the configuration file")
			return
		}
		for _, bucketConfiguration := range config.Credentials {

			couple, err := lib.NewS3BucketCouple(bucketConfiguration)
			if err != nil {
				log.LogE(err).Error("Error in generating the Couple of Buckets")
			}

			dataObjects, err := couple.Data.ListObject()
			if err != nil {
				log.LogE(err).Error("Error in listing the object in the bucket",
					couple.Data.BucketName)
			}
			for _, item := range dataObjects.Contents {
				fmt.Println("Name:         ", *item.Key)
				fmt.Println("Last modified:", *item.LastModified)
				fmt.Println("Size:         ", *item.Size)
				fmt.Println("Storage class:", *item.StorageClass)
				fmt.Println("")
			}

			statusObjects, err := couple.Status.ListObject()
			if err != nil {
				log.LogE(err).Error("Error in listing the object in the bucket",
					couple.Status.BucketName)
			}
			for _, item := range statusObjects.Contents {
				fmt.Println("Name:         ", *item.Key)
				fmt.Println("Last modified:", *item.LastModified)
				fmt.Println("Size:         ", *item.Size)
				fmt.Println("Storage class:", *item.StorageClass)
				fmt.Println("")
			}
		}

	},
}
