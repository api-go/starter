package starter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/api-go/log"
)

func init() {
	for _, name := range []string{"log", "logv"} {
		description := "Follow the configured log file (-tail 100 -f) with built-in formatting"
		if name == "logv" {
			description = "Format logs from stdin or files (-json for JSON output)"
		}
		AddCommand(name, description, func() {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			if err := log.RunCommand(ctx, name, os.Args[2:], os.Stdin, os.Stdout); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, syscall.EPIPE) {
				fmt.Fprintln(os.Stderr, name+":", err)
				os.Exit(1)
			}
		})
	}
}
