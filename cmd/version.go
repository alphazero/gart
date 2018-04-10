// Doost!

package main

import (
	"context"
	"fmt"
	"go/build"
	"os"
)

func parseVersionArgs(args []string) (Command, Option, error) {
	return versionCommand, nil, nil
}

func versionCommand(context.Context, Option) error {
	hos := build.Default.GOOS
	arch := build.Default.GOARCH
	fmt.Fprintf(os.Stdout,
		"gart - the glorious archive tool - version alpha.0.0 (%s %s)\n",
		hos, arch)
	return nil
}
