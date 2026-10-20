package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"devlok/internal/ports"
)

func newPortsCommand(jsonOut *bool) *cobra.Command {
	var extra []int
	cmd := &cobra.Command{
		Use:   "ports",
		Short: "Periksa port dan proses pemiliknya",
		RunE: func(cmd *cobra.Command, args []string) error {
			list := append([]int{80, 443, 3306, 8080, 8443}, extra...)
			statuses := ports.CheckAll(list)
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "ports", true, statuses)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "PORT\tSTATUS\tPID\tPROCESS\tPATH")
			for _, st := range statuses {
				state := "free"
				if !st.Available {
					state = "in use"
				}
				fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\n", st.Port, state, st.PID, st.Process, st.Path)
			}
			return tw.Flush()
		},
	}
	cmd.Flags().IntSliceVar(&extra, "port", nil, "Port tambahan yang diperiksa")
	return cmd
}
