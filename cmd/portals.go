package cmd

import (
	"fmt"
	"strings"
	"sync"

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

		var wg sync.WaitGroup
		for _, bucketConfiguration := range config.Credentials {

			couple, err := lib.NewS3BucketCouple(bucketConfiguration)
			repo := cvmfs.NewRepo(bucketConfiguration.CVMFSRepo)
			if err != nil {
				log.LogE(err).Error("Error in generating the Couple of Buckets")
				continue
			}

			wg.Add(1)
			go func() {

				for {
					inputChan, outputChan := lib.NewPipeline()

					b := couple.Data
					objectChan := make(chan s3.Object, 10)

					b.SpoolAllObject(nil, objectChan)

					go func() {
						for object := range objectChan {
							fmt.Println(*object.Key)
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
						close(inputChan)
					}()

					for output := range outputChan {
						fmt.Println(output)
					}
				}
			}()

		}
		wg.Wait()

	},
}
