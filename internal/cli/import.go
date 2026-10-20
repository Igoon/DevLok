package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/importlaragon"
	"devlok/internal/platform"
	"devlok/internal/projects"
)

func newImportCommand(jsonOut *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Migrasi project dari tool lain",
	}
	cmd.AddCommand(newImportLaragonCommand(jsonOut))
	return cmd
}

func newImportLaragonCommand(jsonOut *bool) *cobra.Command {
	var dryRun, yes, adopt bool
	var only string
	cmd := &cobra.Command{
		Use:   "laragon <path>",
		Short: "Impor project dari instalasi Laragon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sites, err := importlaragon.Plan(args[0])
			if err != nil {
				return err
			}
			if only != "" {
				filtered := sites[:0]
				for _, s := range sites {
					if s.Name == only {
						filtered = append(filtered, s)
					}
				}
				sites = filtered
			}
			write := yes && !dryRun
			results, err := importlaragon.Import(sites, !write)
			if err != nil {
				return err
			}
			if write && adopt {
				state, err := platform.Current().StateDir()
				if err == nil {
					for _, r := range results {
						if r.Action == "create" || r.Action == "skip" {
							_ = projects.Add(state, r.Site.ProjectDir)
						}
					}
				}
			}
			failed := false
			for _, r := range results {
				if r.Action == "error" {
					failed = true
				}
			}
			if *jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), "import.laragon", !failed, map[string]any{"results": results, "written": write}); err != nil {
					return err
				}
			} else {
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
				fmt.Fprintln(tw, "NAME\tDOMAIN\tROOT\tSOURCE\tACTION")
				for _, r := range results {
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Site.Name, r.Site.Domain, r.Site.Root, r.Site.Source, r.Action)
				}
				_ = tw.Flush()
				fmt.Fprintln(cmd.OutOrStdout(), importlaragon.Summary(results))
				if !write {
					fmt.Fprintln(cmd.OutOrStdout(), "Belum ada perubahan. Jalankan dengan --yes untuk menulis manifest.")
				}
			}
			if failed {
				return devlokerr.New(devlokerr.CodeStateInconsistent, "Sebagian project gagal diimpor.", "Periksa output di atas.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Tampilkan rencana tanpa menulis")
	cmd.Flags().BoolVar(&yes, "yes", false, "Tulis manifest")
	cmd.Flags().StringVar(&only, "site", "", "Impor satu site saja")
	cmd.Flags().BoolVar(&adopt, "adopt", false, "Daftarkan project ke GUI setelah impor")
	return cmd
}
