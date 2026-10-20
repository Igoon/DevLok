package cli

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"devlok/internal/doctor"
)

func newDoctorCommand(jsonOut *bool) *cobra.Command {
	var manifestPath string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Periksa lingkungan dan dependency",
		RunE: func(cmd *cobra.Command, args []string) error {
			rep := doctor.Run(context.Background(), doctor.Options{ManifestPath: manifestPath})
			if *jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), "doctor", rep.OK(), rep); err != nil {
					return err
				}
			} else {
				printReport(cmd, rep)
			}
			if !rep.OK() {
				return exitError{code: 1}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&manifestPath, "manifest", "", "Path .devlok.yaml")
	return cmd
}

func printReport(cmd *cobra.Command, rep doctor.Report) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "DevLok doctor %s (%s)\n\n", rep.Version, rep.Platform)
	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	for _, c := range rep.Checks {
		fmt.Fprintf(tw, "[%s]\t%s\t%s\n", c.Status, c.Name, c.Message)
	}
	_ = tw.Flush()

	okCount, warnCount, failCount := rep.Counts()
	fmt.Fprintf(out, "\n%d ok, %d warn, %d fail\n", okCount, warnCount, failCount)

	printedHint := false
	for _, c := range rep.Checks {
		if c.Hint == "" || c.Status == doctor.StatusOK {
			continue
		}
		if !printedHint {
			fmt.Fprintln(out, "\nSaran:")
			printedHint = true
		}
		fmt.Fprintf(out, "- %s: %s\n", c.Name, c.Hint)
	}
}
