package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"devlok/internal/version"
)

func newVersionCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Tampilkan versi",
		RunE: func(cmd *cobra.Command, args []string) error {
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "version", true, version.Current())
			}
			fmt.Fprintln(cmd.OutOrStdout(), version.String())
			return nil
		},
	}
}
