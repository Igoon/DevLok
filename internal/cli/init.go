package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"devlok/internal/config"
)

func newInitCommand(jsonOut *bool) *cobra.Command {
	var opts config.InitOptions
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Buat manifest .devlok.yaml",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			opts.Dir = dir
			path, m, err := config.Init(opts)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "init", true, map[string]any{
					"path":     path,
					"manifest": m,
				})
			}
			out := cmd.OutOrStdout()
			display := path
			if rel, err := filepath.Rel(dir, path); err == nil {
				display = rel
			}
			fmt.Fprintf(out, "Manifest dibuat: %s\n", display)
			fmt.Fprintf(out, "Project: %s\n", m.Name)
			fmt.Fprintf(out, "Root: %s\n", m.Web.Root)
			url := "http://" + m.BaseDomain()
			if m.HTTPSEnabled() {
				url = "https://" + m.BaseDomain()
				if m.Ports.HTTPS > 0 && m.Ports.HTTPS != 443 {
					url += fmt.Sprintf(":%d", m.Ports.HTTPS)
				}
			} else if m.Ports.HTTP > 0 && m.Ports.HTTP != 80 {
				url += fmt.Sprintf(":%d", m.Ports.HTTP)
			}
			fmt.Fprintf(out, "URL: %s\n", url)
			fmt.Fprintln(out, "Langkah berikut: dl doctor")
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Name, "name", "", "Nama project")
	cmd.Flags().StringVar(&opts.Root, "root", "", "Document root relatif")
	cmd.Flags().StringVar(&opts.TLD, "tld", "", "TLD lokal (default test)")
	cmd.Flags().StringVar(&opts.PHP, "php", "", "Versi PHP (default 8.4)")
	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Domain utama, contoh: app.local")
	cmd.Flags().IntVar(&opts.HTTPPort, "http-port", 0, "Port HTTP (0 = otomatis)")
	cmd.Flags().IntVar(&opts.HTTPSPort, "https-port", 0, "Port HTTPS (0 = otomatis)")
	cmd.Flags().BoolVar(&opts.NoDatabase, "no-db", false, "Jangan aktifkan MySQL/phpMyAdmin")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Timpa manifest yang ada")
	return cmd
}
