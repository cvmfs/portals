package lib

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/siscia/portals/log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3CredentialsProvider struct {
	accessKey string
	secretKey string
}

func NewS3CredentialProvider(accessKey, secretKey string) (S3CredentialsProvider, error) {
	if accessKey == "" {
		return S3CredentialsProvider{}, fmt.Errorf("AccessKey not provided")
	}
	if secretKey == "" {
		return S3CredentialsProvider{}, fmt.Errorf("SecretKey not provided")
	}
	return S3CredentialsProvider{accessKey: accessKey, secretKey: secretKey}, nil
}

func (s S3CredentialsProvider) Retrieve() (credentials.Value, error) {
	return credentials.Value{
		AccessKeyID:     s.accessKey,
		SecretAccessKey: s.secretKey,
		SessionToken:    "",
		ProviderName:    ""}, nil
}

func (s S3CredentialsProvider) IsExpired() bool { return false }

type S3Bucket struct {
	BucketName string
	Session    session.Session
}

func NewS3Bucket(bucketName, region, hostURL, accessKey, secretKey string) (bucket S3Bucket, err error) {
	S3CredentialsProvide, err := NewS3CredentialProvider(accessKey, secretKey)
	if err != nil {
		err = fmt.Errorf("Error in generating the CredentialProvider: %s", err)
		return
	}
	credentials := credentials.NewCredentials(S3CredentialsProvide)
	sess, err := session.NewSession(
		aws.NewConfig().
			WithCredentials(credentials).
			WithRegion(region).
			WithEndpoint(hostURL))
	if err != nil {
		err = fmt.Errorf("Error in creating S3 session: %s", err)
		return
	}
	bucket.BucketName = bucketName
	bucket.Session = *sess
	return
}

func (b S3Bucket) ListObject() (*s3.ListObjectsOutput, error) {
	sess := s3.New(&b.Session)
	return sess.ListObjects(&s3.ListObjectsInput{Bucket: &b.BucketName})
}

type S3BucketCouple struct {
	Data   S3Bucket
	Status S3Bucket
}

func NewS3BucketCouple(bc BucketConfiguration) (couple S3BucketCouple, err error) {
	region := bc.Region
	hostURL := bc.HostURL
	accessKey := bc.AccessKey
	secretKey := bc.SecretKey
	data, err := NewS3Bucket(bc.Bucket, region, hostURL, accessKey, secretKey)
	if err != nil {
		err = fmt.Errorf("Error in generating the structure for the data bucket: %s | %s",
			bc.Bucket, err)
		return
	}
	status, err := NewS3Bucket(bc.StatusBucket, region, hostURL, accessKey, secretKey)
	if err != nil {
		err = fmt.Errorf("Error in generating the structure for the status bucket: %s | %s",
			bc.StatusBucket, err)
		return
	}
	couple.Data = data
	couple.Status = status
	return
}

func UploadPingToStatusBucket(s3c S3BucketCouple) {
	status := s3c.Status
	uploader := s3manager.NewUploader(&status.Session)
	l := log.Decorate(map[string]string{
		"Action":        "PING",
		"Status Bucket": status.BucketName,
	})
	for {
		err := func() error {
			t := time.Now()
			timestamp := fmt.Sprint(t)

			body := bytes.NewBuffer(make([]byte, 0))
			body.WriteString(timestamp)

			_, err := uploader.Upload(&s3manager.UploadInput{
				Bucket:      aws.String(status.BucketName),
				Key:         aws.String("PING"),
				ContentType: aws.String("text"),
				Body:        body,
			})
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			l(log.LogE(err)).Error("Error in PINGing the status bucket")
		} else {
			l(log.Log()).Info("Successfully PINGing the status bucket")
		}
		time.Sleep(30 * time.Second)
	}
}

type S3File struct {
	downloader             *s3manager.Downloader
	remotePath             *s3.GetObjectInput
	tempPath               string
	err                    error
	downloadedWithoutError bool
}

func NewS3File(downloader *s3manager.Downloader, remotePath *s3.GetObjectInput) S3File {
	return S3File{
		downloader: downloader,
		remotePath: remotePath,
		tempPath:   "",
		err:        nil,
		downloadedWithoutError: false,
	}
}

func (s3f S3File) Download() {
	f, err := ioutil.TempFile("", "portal")
	if err != nil {
		s3f.err = err
		return
	}
	_, err = s3f.downloader.Download(f, s3f.remotePath)
	if err != nil {
		s3f.err = err
		return
	}
	s3f.downloadedWithoutError = true
	return
}

func (s3f S3File) Error() error {
	return s3f.err
}

func (s3f S3File) TemporaryLocation() string {
	if s3f.downloadedWithoutError {
		return s3f.tempPath
	}
	if s3f.downloadedWithoutError == false && s3f.Error() == nil {
		s3f.Download()
		return s3f.TemporaryLocation()
	}
	return ""
}

func (s3f S3File) Clean() error {
	if s3f.downloadedWithoutError == false {
		return nil
	}
	err := os.Remove(s3f.tempPath)
	return err
}
