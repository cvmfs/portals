package cmd

import (
	"fmt"
	"strings"

	"github.com/cvmfs/portals/cvmfs"
	"github.com/cvmfs/portals/lib"
	"github.com/cvmfs/portals/log"

	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(portalsCmd)
}

var portalsCmd = &cobra.Command{
	Use:   "portals",
	Short: "Start the portals",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, arg []string) {
		config, err := lib.ParseConfig(arg[0])
		if err != nil {
			log.LogE(err).Error("Error in parsing the configuration file")
			return
		}

		inputChan, outputChan := lib.NewPipeline()

		for _, bucketConfiguration := range config.Credentials {

			couple, err := lib.NewS3BucketCouple(bucketConfiguration)
			if err != nil {
				log.LogE(err).Error("Error in generating the Couple of Buckets")
				continue
			}

			go func() {
				repo := cvmfs.NewRepo(bucketConfiguration.CVMFSRepo)
				for {
					b := couple.Data
					objectChan := make(chan s3.Object, 10)

					b.SpoolAllObject(nil, objectChan)

					for object := range objectChan {
						keySplitted := strings.Split(*object.Key, ".")
						if keySplitted[len(keySplitted)-1] == "tar" {
							s3o := lib.NewS3Object(
								couple.Data.BucketName,
								couple.Status.BucketName,
								object,
								&couple.Status.Session,
								&repo)
							inputChan <- s3o
						}
					}

				}
			}()

		}

		for output := range outputChan {

			fmt.Println(output)
		}
	},
}
