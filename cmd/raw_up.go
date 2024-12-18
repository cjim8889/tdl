package cmd

import (
	"context"

	"github.com/gotd/td/telegram"
	"github.com/spf13/cobra"

	"github.com/iyear/tdl/app/raw_up"
	"github.com/iyear/tdl/core/storage"
)

func NewRawUpload() *cobra.Command {
	var opts raw_up.Options

	cmd := &cobra.Command{
		Use:     "rupload",
		Aliases: []string{"rup"},
		Short:   "Upload anything to Telegram Without Sending",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tRun(cmd.Context(), func(ctx context.Context, c *telegram.Client, kvd storage.Storage) error {
				return raw_up.Run(ctx, c, kvd, opts)
			})
		},
	}

	const (
		path = "path"
	)
	cmd.Flags().StringVarP(&opts.Path, path, "p", "", "file")
	cmd.Flags().StringVar(&opts.Name, "name", "default", "name")

	// completion and validation
	_ = cmd.MarkFlagRequired(path)

	return cmd
}
