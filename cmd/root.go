package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/coreruleset/albedo/server"
	"github.com/spf13/cobra"
)

func Execute() error {
	rootCmd := NewRootCommand()
	return rootCmd.ExecuteContext(context.Background())
}

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "albedo",
		Short: "HTTP reflector and black hole",
		RunE:  runE,
	}
	rootCmd.PersistentFlags().IntP("port", "p", 8080, "port to listen on")
	rootCmd.PersistentFlags().StringP("bind", "b", "0.0.0.0", "address to bind to")
	rootCmd.PersistentFlags().Bool("debug", false, "Log debugging information")

	return rootCmd
}

func runE(cmd *cobra.Command, _ []string) error {
	port, _ := cmd.Flags().GetInt("port")
	binding, _ := cmd.Flags().GetString("bind")
	debug, _ := cmd.Flags().GetBool("debug")
	if debug {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(handler)
		slog.SetDefault(logger)
	}

	server.Start(binding, port)
	return nil
}
