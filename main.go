package main

import (
	"github.com/siscia/portals/cmd"
	_ "github.com/siscia/portals/cvmfs"
	_ "github.com/siscia/portals/lib"
)

func main() {
	cmd.EntryPoint()
}
