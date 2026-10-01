package main

import (
	"fmt"
	"os"

	"github.com/A1b3rt0M3rcad0/wos/internal/server"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 1 && args[0] == "version" {
		fmt.Println(server.CurrentVersion().String())
		return 0
	}
	if len(args) == 2 && args[0] == "config" && args[1] == "validate" {
		if err := server.DefaultConfig().Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
			return 1
		}
		fmt.Println("configuration valid")
		return 0
	}

	fmt.Fprintln(os.Stderr, "usage: wos <version|config validate>")
	return 2
}
