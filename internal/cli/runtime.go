package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	rt "devlok/internal/runtime"
)

type runtimeRow struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Status   string `json:"status"`
	Platform string `json:"platform"`
}

type verifyRow struct {
	Target string `json:"target"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

func newRuntimeCommand(jsonOut *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Kelola runtime (nginx, php, mysql, node, composer, adminer)",
	}
	cmd.AddCommand(newRuntimeListCommand(jsonOut))
	cmd.AddCommand(newRuntimeAddCommand(jsonOut))
	cmd.AddCommand(newRuntimeRemoveCommand(jsonOut))
	cmd.AddCommand(newRuntimeVerifyCommand(jsonOut))
	return cmd
}

func newRuntimeListCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Daftar runtime tersedia dan terpasang",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newRuntimeManager(false, nil)
			if err != nil {
				return err
			}
			installed, err := mgr.ListInstalled()
			if err != nil {
				return devlokerr.Wrap(devlokerr.CodeStateInconsistent, err, "Daftar runtime terpasang tidak dapat dibaca.", "")
			}
			have := map[string]bool{}
			for _, in := range installed {
				have[in.Name+"@"+in.Version] = true
			}
			key := mgr.OS + "/" + mgr.Arch
			var rows []runtimeRow
			regs := append([]rt.Registry(nil), mgr.Registries...)
			sort.Slice(regs, func(i, j int) bool { return regs[i].Name < regs[j].Name })
			for _, reg := range regs {
				for _, v := range reg.Versions {
					if _, ok := v.Artifacts[key]; !ok {
						continue
					}
					status := "available"
					if have[reg.Name+"@"+v.Version] {
						status = "installed"
					}
					rows = append(rows, runtimeRow{Name: reg.Name, Version: v.Version, Status: status, Platform: key})
				}
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "runtime.list", true, rows)
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tVERSION\tSTATUS\tPATH")
			installedAt := map[string]string{}
			for _, in := range installed {
				installedAt[in.Name+"@"+in.Version] = in.Path
			}
			for _, r := range rows {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.Name, r.Version, r.Status, installedAt[r.Name+"@"+r.Version])
			}
			return tw.Flush()
		},
	}
}

func newRuntimeAddCommand(jsonOut *bool) *cobra.Command {
	var offline bool
	cmd := &cobra.Command{
		Use:   "add <name>[@version]",
		Short: "Unduh, verifikasi, dan pasang runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, version, err := splitTarget(args[0], false)
			if err != nil {
				return err
			}
			mgr, err := newRuntimeManager(offline, runtimeProgress(cmd, *jsonOut))
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
			defer cancel()
			inst, downloaded, err := mgr.Install(ctx, name, version)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "runtime.add", true, map[string]any{
					"runtime":    inst,
					"downloaded": downloaded,
				})
			}
			state := "sudah terpasang"
			if downloaded {
				state = "diunduh"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s@%s %s\nPath: %s\n", inst.Name, inst.Version, state, inst.Path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&offline, "offline", false, "Hanya memakai cache lokal")
	return cmd
}

func newRuntimeRemoveCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>@<version>",
		Short: "Hapus runtime terpasang",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, version, err := splitTarget(args[0], true)
			if err != nil {
				return err
			}
			mgr, err := newRuntimeManager(false, nil)
			if err != nil {
				return err
			}
			if err := mgr.Remove(name, version); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "runtime.remove", true, map[string]string{"removed": name + "@" + version})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Dihapus: %s@%s\n", name, version)
			return nil
		},
	}
}

func newRuntimeVerifyCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "verify [name[@version]]",
		Short: "Verifikasi runtime terpasang dan checksum cache",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newRuntimeManager(false, nil)
			if err != nil {
				return err
			}
			type target struct{ name, version string }
			var targets []target
			if len(args) == 1 {
				name, version, err := splitTarget(args[0], false)
				if err != nil {
					return err
				}
				targets = append(targets, target{name, version})
			} else {
				installed, err := mgr.ListInstalled()
				if err != nil {
					return err
				}
				for _, in := range installed {
					targets = append(targets, target{in.Name, in.Version})
				}
			}
			if len(targets) == 0 {
				if *jsonOut {
					return writeJSON(cmd.OutOrStdout(), "runtime.verify", true, []verifyRow{})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Tidak ada runtime terpasang.")
				return nil
			}
			rows := make([]verifyRow, 0, len(targets))
			failed := false
			for _, t := range targets {
				err := mgr.Verify(t.name, t.version)
				row := verifyRow{Target: t.name + "@" + t.version, OK: err == nil}
				if err != nil {
					row.Error = err.Error()
					failed = true
				}
				rows = append(rows, row)
			}
			if *jsonOut {
				if err := writeJSON(cmd.OutOrStdout(), "runtime.verify", !failed, rows); err != nil {
					return err
				}
			} else {
				for _, r := range rows {
					if r.OK {
						fmt.Fprintf(cmd.OutOrStdout(), "[ok]   %s\n", r.Target)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), "[fail] %s: %s\n", r.Target, r.Error)
					}
				}
			}
			if failed {
				return exitError{code: 1}
			}
			return nil
		},
	}
}

func newRuntimeManager(offline bool, progress func(rt.Progress)) (*rt.Manager, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, devlokerr.Wrap(devlokerr.CodeInternal, err, "State dir tidak dapat ditentukan.", "")
	}
	return rt.NewManager(rt.Options{
		Root:     filepath.Join(state, "runtimes"),
		Cache:    filepath.Join(state, "cache", "downloads"),
		Offline:  offline,
		Progress: progress,
	})
}

func splitTarget(raw string, requireVersion bool) (string, string, error) {
	name, version, found := strings.Cut(raw, "@")
	if name == "" || strings.ContainsAny(name, " /\\") {
		return "", "", devlokerr.New(devlokerr.CodeInvalidArgument, "Target runtime tidak valid: "+raw, "Gunakan format nama@versi, contoh: php@8.4.")
	}
	if !found || version == "" {
		if requireVersion {
			return "", "", devlokerr.New(devlokerr.CodeInvalidArgument, "Versi wajib diisi untuk perintah ini.", "Gunakan format nama@versi, contoh: php@8.4.26.")
		}
		if found && version == "" {
			return "", "", devlokerr.New(devlokerr.CodeInvalidArgument, "Versi kosong pada target: "+raw, "Hilangkan tanda @ atau isi versinya.")
		}
		return name, "", nil
	}
	return name, version, nil
}

func runtimeProgress(cmd *cobra.Command, jsonOut bool) func(rt.Progress) {
	if jsonOut {
		return nil
	}
	errOut := cmd.ErrOrStderr()
	lastPhase := ""
	return func(p rt.Progress) {
		switch p.Phase {
		case "download":
			if p.Total > 0 {
				pct := p.Downloaded * 100 / p.Total
				fmt.Fprintf(errOut, "\r  download %3d%% (%s / %s)   ", pct, humanBytes(p.Downloaded), humanBytes(p.Total))
			} else if lastPhase != "download" {
				fmt.Fprintf(errOut, "  download %s@%s\n", p.Name, p.Version)
			}
		case "verify":
			if lastPhase == "download" {
				fmt.Fprintln(errOut)
			}
			fmt.Fprintln(errOut, "  verify checksum")
		case "extract":
			fmt.Fprintln(errOut, "  extract")
		case "done":
			fmt.Fprintln(errOut, "  selesai")
		}
		lastPhase = p.Phase
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
