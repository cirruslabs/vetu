package main

import (
	"context"
	"fmt"
	"github.com/cirruslabs/vetu/internal/command"
	"github.com/cirruslabs/vetu/internal/version"
	"github.com/getsentry/sentry-go"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	// Initialize Sentry
	var release string

	if version.Version != "unknown" {
		release = fmt.Sprintf("vetu@v%s", version.Version)
	}

	err := sentry.Init(sentry.ClientOptions{
		Release:          release,
		AttachStacktrace: true,
	})
	if err != nil {
		log.Fatalf("failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	// Enrich future events with Cirrus CI-specific tags
	if tags, ok := os.LookupEnv("CIRRUS_SENTRY_TAGS"); ok {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			for _, tag := range strings.Split(tags, ",") {
				splits := strings.SplitN(tag, "=", 2)
				if len(splits) != 2 {
					continue
				}

				scope.SetTag(splits[0], splits[1])
			}
		})
	}

	// Set up a signal-interruptible context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	// Disable log timestamping
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Run the command
	if err := command.NewRootCmd().ExecuteContext(ctx); err != nil {
		// Capture the error into Sentry
		sentry.CaptureException(err)
		sentry.Flush(2 * time.Second)

		// Capture the error into stderr and terminate
		cancel()

		//nolint:gocritic // "log.Fatal will exit, and `defer sentry.Recover()` will not run" — it's OK,
		// since we're already capturing the error above
		log.Fatal(err)
	}

	cancel()
}
