package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server"
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
		if err := server.ConfigFromEnv().Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "invalid configuration: %v\n", err)
			return 1
		}
		fmt.Println("configuration valid")
		return 0
	}
	if len(args) == 1 && args[0] == "server" {
		cfg := server.ConfigFromEnv()
		runtime, err := server.OpenRuntime(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot start WOS: %v\n", err)
			return 1
		}
		defer runtime.Close()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := runtime.Serve(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "WOS server stopped with error: %v\n", err)
			return 1
		}
		return 0
	}

	fmt.Fprintln(os.Stderr, "usage: wos <server|version|config validate>")
	return 2
}
