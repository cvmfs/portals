package main

import (
	/*
		"fmt"
		"os"
	*/

	"github.com/siscia/portals/cmd"
	_ "github.com/siscia/portals/cvmfs"
	_ "github.com/siscia/portals/lib"
	/*
		"github.com/aws/aws-sdk-go/aws"
		"github.com/aws/aws-sdk-go/aws/session"
		"github.com/aws/aws-sdk-go/service/s3"
		"github.com/aws/aws-sdk-go/service/s3/s3manager"
	*/)

func main() {
	cmd.EntryPoint()
}
