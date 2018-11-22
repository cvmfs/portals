package lib

/*
The idea here is to have a pipeline, where:
1. Element enter
2. They are downloaded into local storange
3. They are ingested into CVMFS
4. Local resource are cleaned up

What happen in case of failure?  We can return a structure that implement the
interface of the next pipeline, on this structure we can simply shortcut all
the other call or do the "right thing", re-try? Set a re-try in S3? Whatever!
*/

type PipelineInput interface {
	MakeS3RemoteFile() S3RemoteFile
}

type S3RemoteFile interface {
	DownloadFile() S3LocalFile
}

type S3LocalFile interface {
	Ingest() S3IngestedFile
}

type S3IngestedFile interface {
	CleanUp() PipelineOutput
}

type PipelineOutput struct{}

func NewPipeline() (chan<- PipelineInput, <-chan PipelineOutput) {
	// Each channel has a buffer of $buffer, and we have $workers running
	// at the same time, it means that in the worst case there are $buffer
	// + $workers job on the fly.
	buffer := 10
	workers := 10
	chanInput := make(chan PipelineInput, buffer)
	chanOutput := make(chan PipelineOutput, buffer)

	go func() {
		downloadChan := make(chan S3RemoteFile, buffer)
		ingestChan := make(chan S3LocalFile, buffer)
		cleanupChan := make(chan S3IngestedFile, buffer)

		for w := 1; w <= workers; w++ {
			go func() {
				for pipelineInput := range chanInput {
					remoteFileToDownload :=
						pipelineInput.MakeS3RemoteFile()
					downloadChan <- remoteFileToDownload
				}
			}()

			go func() {
				for s3RemoteFile := range downloadChan {
					localFileToIngest := s3RemoteFile.DownloadFile()
					ingestChan <- localFileToIngest
				}
			}()

			go func() {
				for s3LocalFile := range ingestChan {
					ingestedFileToCleanup := s3LocalFile.Ingest()
					cleaupChan <- ingestedFileToCleanup
				}
			}()

			go func() {
				for s3IngestedFile := range cleanupChan {
					cleanedupFileToReturn := s3IngestedFile.Cleaup()
					chanOutput <- cleanedupFileToReturn
				}
			}()
		}
	}()

	return chanInput, chanOutput
}
