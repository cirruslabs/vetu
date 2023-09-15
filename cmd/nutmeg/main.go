package main

import (
	"context"
	"github.com/cirruslabs/nutmeg/internal/command"
	"log"
	"os"
	"os/signal"
)

func main() {
	// Set up a signal-interruptible context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// Run the command
	if err := command.NewRootCmd().ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}
