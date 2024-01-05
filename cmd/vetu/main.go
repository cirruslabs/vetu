package main

import (
	"context"
	"github.com/cirruslabs/vetu/internal/command"
	"log"
	"os"
	"os/signal"
)

func main() {
	// Set up a signal-interruptible context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	// Disable log timestamping
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	// Run the command
	if err := command.NewRootCmd().ExecuteContext(ctx); err != nil {
		cancel()
		log.Fatal(err)
	}

	cancel()
}
