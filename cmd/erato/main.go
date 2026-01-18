// Package main is the erato CLI executable
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/stolasapp/erato/internal/command"
)

func main() { os.Exit(run()) }

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := command.RootCommand().ExecuteContext(ctx)
	if err != nil {
		return 1
	}
	return 0
}
