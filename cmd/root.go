package cmd

import (
	"context"

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
	port := new(int)
	binding := new(string)
	rootCmd.PersistentFlags().IntVarP(port, "port", "p", 8080, "port to listen on")
	rootCmd.PersistentFlags().StringVarP(binding, "bind", "b", "0.0.0.0", "address to bind to")

	return rootCmd
}

func runE(cmd *cobra.Command, _ []string) error {
	port, _ := cmd.Flags().GetInt("port")
	binding, _ := cmd.Flags().GetString("bind")

	_ = server.Start(binding, port)

	return nil
}
