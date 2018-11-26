package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cvmfs/portals/cvmfs"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

/*
The idea here is to have a pipeline, where:
1. Element enter
2. They are downloaded into local storange
3. They are ingested into CVMFS
4. Local resource are cleaned up
5. Element exits

What happen in case of failure?  We can return a structure that implement the
interface of the next pipeline, on this structure we can simply shortcut all
the other call or do the "right thing", re-try? Set a re-try in S3? Whatever!
*/

type PipelineInput interface {
	MakeS3RemoteFile() IS3RemoteFile
}

type IS3RemoteFile interface {
	DownloadFile() IS3LocalFile
}

type IS3LocalFile interface {
	Ingest() IS3IngestedFile
}

type IS3IngestedFile interface {
	Cleanup() PipelineOutput
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
		downloadChan := make(chan IS3RemoteFile, buffer)
		ingestChan := make(chan IS3LocalFile, buffer)
		cleanupChan := make(chan IS3IngestedFile, buffer)

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
					cleanupChan <- ingestedFileToCleanup
				}
			}()

			go func() {
				for s3IngestedFile := range cleanupChan {
					cleanedupFileToReturn := s3IngestedFile.Cleanup()
					chanOutput <- cleanedupFileToReturn
				}
			}()
		}
	}()

	return chanInput, chanOutput
}

// The part above is the very important part that is necessary to understand
// before to dive into the real code, what is below is more details
// implementation

type GenericError struct{}

func (e GenericError) DownloadFile() IS3LocalFile {
	return GenericError{}
}

func (e GenericError) Ingest() IS3IngestedFile {
	return GenericError{}
}

func (e GenericError) Cleanup() PipelineOutput {
	return PipelineOutput{}
}

type S3Object struct {
	bucket       string
	statusBucket string
	key          string
	hash         string
	session      *session.Session
	cvmfsRepo    *cvmfs.Repo
}

func (s3o S3Object) UploadStatus(status string) error {
	t := time.Now()
	timestamp := fmt.Sprint(t)
	body := bytes.NewBuffer(make([]byte, 0))
	body.WriteString(timestamp)

	key := fmt.Sprintf("%s.%s.%s", s3o.key, s3o.hash, status)

	uploader := s3manager.NewUploader(s3o.session)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3o.statusBucket),
		Key:    aws.String(key),
		Body:   body,
	})
	if err != nil {
		return err
	}
	return nil
}

func NewS3Object(bucket, statusBucket string, s3obj s3.Object, session *session.Session, cvmfsRepo *cvmfs.Repo) S3Object {
	toHash := []byte(fmt.Sprintf("%s%d", *s3obj.Key, s3obj.LastModified.Unix()))
	hash := fmt.Sprintf("%s", sha256.Sum256(toHash))[0:10]
	return S3Object{
		bucket:       bucket,
		statusBucket: statusBucket,
		key:          *s3obj.Key,
		hash:         hash,
		session:      session,
		cvmfsRepo:    cvmfsRepo}
}

func (s3obj S3Object) MakeS3RemoteFile() IS3RemoteFile {
	return s3obj
}

type S3LocalFile struct {
	S3Object
	tempPath string
}

func (s3obj S3Object) DownloadFile() IS3LocalFile {

	f, _ := ioutil.TempFile("", "s3temp")

	defer f.Close()

	go s3obj.UploadStatus("DOWNLOADING")

	downloader := s3manager.NewDownloader(s3obj.session)
	downloader.Download(f, &s3.GetObjectInput{
		Bucket: &s3obj.bucket,
		Key:    &s3obj.key,
	})

	return S3LocalFile{s3obj, f.Name()}
}

type S3IngestedFile struct {
	S3Object
	tempPath string
}

func (s3local S3LocalFile) Ingest() IS3IngestedFile {
	keyPaths := strings.Split(s3local.key, "/")
	cvmfsPath := filepath.Join(keyPaths[:len(keyPaths)-1]...)
	repo := s3local.cvmfsRepo.Name

	go s3local.UploadStatus("INGESTING")

	s3local.cvmfsRepo.Lock.Lock()
	defer s3local.cvmfsRepo.Lock.Unlock()

	cvmfs.ExecCommand("cvmfs_server", "ingest",
		"-t", s3local.tempPath,
		"-b", cvmfsPath,
		repo).Start()

	return S3IngestedFile{s3local.S3Object, s3local.tempPath}
}

func (s3ingested S3IngestedFile) Cleanup() PipelineOutput {
	os.Remove(s3ingested.tempPath)

	go s3ingested.UploadStatus("DELETING")

	return PipelineOutput{}
}
