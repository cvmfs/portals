package lib

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type BucketConfiguration struct {
	CVMFSRepo    string `toml:"cvmfs-repo"`
	AccessKey    string `toml:"access-key"`
	SecretKey    string `toml:"secret-key"`
	Bucket       string `toml:"bucket"`
	StatusBucket string `toml:"status-bucket"`
	HostURL      string `toml:"host-url"`
	Region       string `region:"region"`
}

type Config struct {
	Credentials []BucketConfiguration `toml:"credentials"`
}

func ParseConfig(path string) (config Config, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	err = toml.Unmarshal(bytes, &config)
	if err != nil {
		return
	}
	for i, bucketConfig := range config.Credentials {
		if bucketConfig.StatusBucket == "" {
			config.Credentials[i].StatusBucket = bucketConfig.Bucket + ".status"
		}
		if bucketConfig.Region == "" {
			config.Credentials[i].Region = "us-east-1"
		}
	}
	return
}
