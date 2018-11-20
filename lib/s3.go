package lib

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"
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
