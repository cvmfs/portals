package lib

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type BucketConfiguration struct {
	AccessKey    string `toml:"access-key"`
	SecretKey    string `toml:"secret-key"`
	Bucket       string `toml:"bucket"`
	StatusBucket string `toml:"status-bucket"`
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
	return
}
