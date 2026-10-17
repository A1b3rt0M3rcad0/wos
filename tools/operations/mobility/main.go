// Command mobility is an optional source-tree wrapper for the same operator
// implementation shipped in the native `wos storage` command family.
package main

import (
	"context"
	"os"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/operations/mobility"
)

func main() {
	os.Exit(mobility.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
