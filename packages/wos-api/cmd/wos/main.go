package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 4 && args[0] == "db" && args[1] == "restore" {
		if err := sqlite.RestoreFile(args[2], args[3]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("restore completed into a new SQLite file")
		return 0
	}
	if len(args) >= 2 && args[0] == "db" && (args[1] == "migrate" || (args[1] == "backup" && len(args) == 3)) {
		cfg := server.ConfigFromEnv()
		cfg.Storage.MigrateOnStart = args[1] == "migrate"
		runtime, err := server.OpenRuntime(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer runtime.Close()
		if args[1] == "migrate" {
			err = runtime.Migrate(context.Background())
		} else {
			err = runtime.Backup(context.Background(), args[2])
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("database operation completed")
		return 0
	}
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
	if len(args) == 2 && args[0] == "mcp" && args[1] == "stdio" {
		cfg := server.ConfigFromEnv()
		cfg.MCP.Enabled = true
		if cfg.Auth.Mode != server.AuthModeLocal {
			fmt.Fprintln(os.Stderr, "stdio requires local authentication; use Streamable HTTP for remote tokens")
			return 1
		}
		runtime, err := server.OpenRuntime(cfg)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer runtime.Close()
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err = runtime.RunStdio(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
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

	fmt.Fprintln(os.Stderr, "usage: wos <server|version|config validate|mcp stdio|db migrate|db backup FILE|db restore SOURCE DESTINATION>")
	return 2
}
