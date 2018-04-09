// Doost!

package main

import (
	"fmt"
	"go/build"
	"os"
)

func (cmd *Cmd) versionCmd([]string) error {
	cmd.run = versionCommand
	return nil
}

func versionCommand() error {
	hos := build.Default.GOOS
	arch := build.Default.GOARCH
	fmt.Fprintf(os.Stdout,
		"gart - the glorious archive tool - version alpha.0.0 (%s %s)\n",
		hos, arch)
	return nil
}
