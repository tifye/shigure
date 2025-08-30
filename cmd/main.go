package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	Execute(ctx)
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shigure",
		Short: "Shigure's CLI tool",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	return cmd
}

func Execute(ctx context.Context) {
	root := newRootCommand()
	if err := root.ExecuteContext(ctx); err != nil {
		panic(err)
	}
}
