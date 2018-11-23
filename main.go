package main

import (
	"github.com/cvmfs/portals/cmd"
	_ "github.com/cvmfs/portals/cvmfs"
	_ "github.com/cvmfs/portals/lib"
)

func main() {
	cmd.EntryPoint()
}
