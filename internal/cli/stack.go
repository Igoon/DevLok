package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/config"
	"devlok/internal/devlokerr"
	"devlok/internal/editor"
	"devlok/internal/hosts"
	"devlok/internal/platform"
	"devlok/internal/ports"
	"devlok/internal/privileged"
	"devlok/internal/projects"
	st "devlok/internal/stack"
)

func newUpCommand(jsonOut *bool) *cobra.Command {
	var projectDir string
	var skipHosts, skipTrust bool
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Siapkan runtime, config, TLS, hosts, dan jalankan service",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
			defer cancel()
			res, err := st.Up(ctx, st.Options{
				ProjectDir: projectDir,
				SkipHosts:  skipHosts,
				SkipTrust:  skipTrust,
				Progress:   runtimeProgress(cmd, *jsonOut),
				Registries: loadRemoteRegistries(""),
			})
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "up", true, res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DevLok up: %s\n", res.Domain)
			fmt.Fprintf(cmd.OutOrStdout(), "URL: %s\n", res.URL)
			if res.PHPMyAdminURL != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "phpMyAdmin: %s\n", res.PHPMyAdminURL)
			}
			if res.DBName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Database: %s (user %s, port %d)\n", res.DBName, res.DBUser, res.DBPort)
				fmt.Fprintln(cmd.OutOrStdout(), "Kredensial: dl db creds "+res.DBName)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Local: %s\n", res.LocalURL)
			for _, w := range res.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&projectDir, "path", "", "Folder project (default: cari manifest)")
	cmd.Flags().BoolVar(&skipHosts, "skip-hosts", false, "Jangan ubah file hosts")
	cmd.Flags().BoolVar(&skipTrust, "skip-trust", false, "Jangan percayai CA lokal")
	return cmd
}

func newDownCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Hentikan service stack",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := st.Down(""); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "down", true, map[string]string{"status": "stopped"})
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Stack dihentikan.")
			return nil
		},
	}
}

func newStatusCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Status stack dan service",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, stateErr := st.LoadState("")
			services, svcErr := st.Services("")
			if svcErr != nil {
				return svcErr
			}
			if stateErr != nil {
				if *jsonOut {
					return writeJSON(cmd.OutOrStdout(), "status", true, map[string]any{"running": false, "services": services})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Stack belum berjalan.")
				return nil
			}
			health := map[string]ports.Status{
				"http":  ports.Check(state.HTTPPort),
				"https": ports.Check(state.HTTPSPort),
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "status", true, map[string]any{
					"running":  true,
					"stack":    state,
					"services": services,
					"ports":    health,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project: %s (%s)\n", state.Name, state.ProjectDir)
			fmt.Fprintf(cmd.OutOrStdout(), "URL: %s\n", state.URL)
			fmt.Fprintf(cmd.OutOrStdout(), "Local: %s\n", state.LocalURL)
			if state.DBName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Database: %s (user %s, port %d)\n", state.DBName, state.DBUser, state.DBPort)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "SERVICE\tSTATE\tPID\tRESTARTS\tERROR")
			for _, s := range services {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\n", s.Name, s.State, s.PID, s.Restarts, s.LastError)
			}
			_ = tw.Flush()
			fmt.Fprintf(cmd.OutOrStdout(), "HTTP %d: %s; HTTPS %d: %s\n", state.HTTPPort, healthText(health["http"]), state.HTTPSPort, healthText(health["https"]))
			return nil
		},
	}
}

func healthText(s ports.Status) string {
	if s.Available {
		return "free"
	}
	return "listening"
}

func newEditCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [path]",
		Short: "Buka file di editor teks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			} else {
				cwd, _ := os.Getwd()
				if found, ok := config.Find(cwd); ok {
					target = found
				} else {
					target = cwd
				}
			}
			abs, err := filepath.Abs(target)
			if err != nil {
				return err
			}
			if _, err := os.Stat(abs); err != nil {
				return devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "File tidak ditemukan: "+abs, "")
			}
			if err := editor.Open(abs); err != nil {
				return devlokerr.Wrap(devlokerr.CodeDependencyMissing, err, "Editor tidak dapat dibuka.", "Instal VS Code atau Notepad++.")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Membuka:", abs)
			return nil
		},
	}
}

func newSiteCommand(jsonOut *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "site",
		Short: "Kelola project di workspace",
	}
	cmd.AddCommand(newSiteCreateCommand(jsonOut))
	cmd.AddCommand(newSiteListCommand(jsonOut))
	cmd.AddCommand(newSiteAdoptCommand(jsonOut))
	cmd.AddCommand(newSiteRemoveCommand(jsonOut))
	return cmd
}

func newSiteCreateCommand(jsonOut *bool) *cobra.Command {
	var workspace, php, domain string
	var httpPort, httpsPort int
	var noDB bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Buat project baru dan jalankan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws := workspace
			if ws == "" {
				var err error
				ws, err = st.WorkspaceDir()
				if err != nil {
					return err
				}
			}
			name := config.SanitizeName(args[0])
			dir := filepath.Join(ws, name)
			if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
				return devlokerr.Wrap(devlokerr.CodePermissionDenied, err, "Folder project tidak dapat dibuat.", "")
			}
			index := filepath.Join(dir, "public", "index.php")
			if _, err := os.Stat(index); os.IsNotExist(err) {
				demo := "<?php\necho '<h1>" + name + "</h1><p>DevLok site aktif.</p>';\n"
				if err := os.WriteFile(index, []byte(demo), 0o644); err != nil {
					return err
				}
			}
			manifestPath := filepath.Join(dir, config.FileName)
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				if _, _, err := config.Init(config.InitOptions{Dir: dir, Name: name, PHP: php, Domain: domain, HTTPPort: httpPort, HTTPSPort: httpsPort, NoDatabase: noDB}); err != nil {
					return err
				}
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
			defer cancel()
			res, err := st.Up(ctx, st.Options{ProjectDir: dir, Progress: runtimeProgress(cmd, *jsonOut), Registries: loadRemoteRegistries("")})
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "site.create", true, res)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Site dibuat: %s\nURL: %s\nPath: %s\n", res.Domain, res.URL, dir)
			for _, w := range res.Warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&workspace, "workspace", "", "Folder workspace")
	cmd.Flags().StringVar(&php, "php", "8.4", "Versi PHP")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain utama")
	cmd.Flags().IntVar(&httpPort, "http-port", 0, "Port HTTP (0 otomatis)")
	cmd.Flags().IntVar(&httpsPort, "https-port", 0, "Port HTTPS (0 otomatis)")
	cmd.Flags().BoolVar(&noDB, "no-db", false, "Jangan buat database")
	return cmd
}

func newSiteListCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Daftar project di workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := st.WorkspaceDir()
			if err != nil {
				return err
			}
			list, err := st.ListProjects(ws)
			if err != nil {
				return err
			}
			seen := map[string]bool{}
			for _, m := range list {
				seen[m.Name] = true
			}
			if state, err := platform.Current().StateDir(); err == nil {
				if extra, err := projects.List(state); err == nil {
					for _, dir := range extra {
						m, err := config.Load(filepath.Join(dir, config.FileName))
						if err != nil || seen[m.Name] {
							continue
						}
						seen[m.Name] = true
						list = append(list, *m)
					}
				}
			}
			sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "site.list", true, list)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tDOMAIN\tROOT\tPHP")
			for _, p := range list {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.Name, p.BaseDomain(), p.Web.Root, p.Web.PHP)
			}
			return tw.Flush()
		},
	}
}

func newSiteRemoveCommand(jsonOut *bool) *cobra.Command {
	var deleteFiles bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Hapus site dari hosts dan certificate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := st.WorkspaceDir()
			if err != nil {
				return err
			}
			name := config.SanitizeName(args[0])
			dir := filepath.Join(ws, name)
			m, err := config.Load(filepath.Join(dir, config.FileName))
			if err != nil {
				return err
			}
			if state, err := st.LoadState(""); err == nil {
				if samePath(state.ProjectDir, dir) {
					if err := st.Down(""); err != nil {
						return err
					}
				}
			}
			plat := platform.Current()
			hm := hosts.New(plat.HostsPath())
			hm.Flush = plat.FlushDNS
			warnings := []string{}
			if _, err := hm.Remove(); err != nil {
				if perr := privileged.RemoveHosts(); perr != nil {
					warnings = append(warnings, "Hosts tidak dapat dibersihkan: "+perr.Error())
				}
			}
			for _, f := range st.CertFiles("", m.Name) {
				_ = os.Remove(f)
			}
			if deleteFiles {
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "site.remove", true, map[string]any{"name": m.Name, "deleted": deleteFiles, "warnings": warnings})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Site dihapus: %s\n", m.Name)
			for _, w := range warnings {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&deleteFiles, "delete", false, "Hapus folder project")
	return cmd
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}

func newSiteAdoptCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "adopt <path>",
		Short: "Daftarkan project di luar workspace ke GUI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, project, err := manifestFor(args[0])
			if err != nil {
				return err
			}
			state, err := platform.Current().StateDir()
			if err != nil {
				return err
			}
			if err := projects.Add(state, project); err != nil {
				return devlokerr.Wrap(devlokerr.CodeInvalidArgument, err, "Project tidak dapat didaftarkan.", "Pastikan ada .devlok.yaml di folder project.")
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "site.adopt", true, map[string]string{"name": m.Name, "path": project})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Project didaftarkan: %s (%s)\n", m.Name, project)
			return nil
		},
	}
}
