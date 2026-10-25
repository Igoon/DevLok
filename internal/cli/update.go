package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	"devlok/internal/stack"
	"devlok/internal/update"
	"devlok/internal/version"
)

func newUpdateCommand(jsonOut *bool) *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Periksa update aplikasi dan runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = check
			mgr, err := newRuntimeManager(false, nil)
			if err != nil {
				return err
			}
			updates, err := update.Check(mgr)
			if err != nil {
				return err
			}
			state, err := platform.Current().StateDir()
			if err != nil {
				return err
			}
			cfg, err := update.LoadConfig(state)
			if err != nil {
				return err
			}
			var appUpdate *update.AppUpdate
			remoteErr := ""
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			appUpdate, err = update.AppUpdateCheck(ctx, cfg)
			if err != nil {
				remoteErr = err.Error()
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "update.check", true, map[string]any{
					"app":          version.Current().Version,
					"updates":      updates,
					"app_update":   appUpdate,
					"remote_error": remoteErr,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DevLok %s\n", version.Current().Version)
			if appUpdate != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Update aplikasi: %s -> %s", appUpdate.Current, appUpdate.Latest)
				if appUpdate.Severity != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " (%s)", appUpdate.Severity)
				}
				fmt.Fprintln(cmd.OutOrStdout())
				if appUpdate.URL != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", appUpdate.URL)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "  Unduh: dl update download")
			}
			if len(updates) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Update runtime tersedia:")
				for _, u := range updates {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s -> %s\n", u.Name, u.Installed, u.Latest)
				}
				if platform.Current().Name() == "darwin" {
					fmt.Fprintln(cmd.OutOrStdout(), "\nService: brew upgrade <nama>. Tool: dl runtime add <nama>@<versi>.")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "\nPasang versi baru dengan: dl runtime add <nama>@<versi>")
				}
			}
			if remoteErr != "" {
				fmt.Fprintf(cmd.ErrOrStderr(), "peringatan: update aplikasi tidak dapat diperiksa: %s\n", remoteErr)
			}
			if platform.Current().Name() == "darwin" && len(updates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Service: brew upgrade nginx php mysql. Tool (Mailpit/phpMyAdmin/Adminer): dl runtime add <nama>@<versi>.")
			} else if appUpdate == nil && len(updates) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Semua runtime terpasang sudah memakai versi terbaru dari registry.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Hanya periksa, jangan pasang")
	cmd.AddCommand(newUpdateDownloadCommand(jsonOut))
	cmd.AddCommand(newUpdateApplyCommand(jsonOut))
	cmd.AddCommand(newUpdateRollbackCommand(jsonOut))
	return cmd
}

func newUpdateDownloadCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "download",
		Short: "Unduh dan verifikasi update aplikasi",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := platform.Current().StateDir()
			if err != nil {
				return err
			}
			cfg, err := update.LoadConfig(state)
			if err != nil {
				return err
			}
			var appUpdate *update.AppUpdate
			checkCtx, cancelCheck := context.WithTimeout(cmd.Context(), 15*time.Second)
			appUpdate, err = update.AppUpdateCheck(checkCtx, cfg)
			cancelCheck()
			if err != nil {
				return err
			}
			if appUpdate == nil {
				if *jsonOut {
					return writeJSON(cmd.OutOrStdout(), "update.download", true, map[string]string{"status": "up-to-date"})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Tidak ada update aplikasi.")
				return nil
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer cancel()
			path, err := update.Download(ctx, appUpdate, filepath.Join(state, "cache", "updates"))
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "update.download", true, map[string]string{
					"version": appUpdate.Latest,
					"file":    path,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Update %s diunduh dan terverifikasi:\n%s\n", appUpdate.Latest, path)
			return nil
		},
	}
}

func newUpdateApplyCommand(jsonOut *bool) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Jalankan installer update aplikasi",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return devlokerr.New(devlokerr.CodeCanceled, "Konfirmasi diperlukan untuk menjalankan installer.", "Tambahkan --yes setelah yakin.")
			}
			state, err := platform.Current().StateDir()
			if err != nil {
				return err
			}
			cfg, err := update.LoadConfig(state)
			if err != nil {
				return err
			}
			var appUpdate *update.AppUpdate
			checkCtx, cancelCheck := context.WithTimeout(cmd.Context(), 15*time.Second)
			appUpdate, err = update.AppUpdateCheck(checkCtx, cfg)
			cancelCheck()
			if err != nil {
				return err
			}
			if appUpdate == nil {
				if *jsonOut {
					return writeJSON(cmd.OutOrStdout(), "update.apply", true, map[string]string{"status": "up-to-date"})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Tidak ada update aplikasi.")
				return nil
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer cancel()
			path, err := update.Download(ctx, appUpdate, filepath.Join(state, "cache", "updates"))
			if err != nil {
				return err
			}
			if _, _, err := update.InstallerCommand(path); err != nil {
				return err
			}
			if err := stack.Down(""); err != nil {
				return err
			}
			if err := update.LaunchInstaller(path); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "update.apply", true, map[string]string{
					"version": appUpdate.Latest,
					"file":    path,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installer %s dijalankan. Selesaikan instalasi, lalu buka DevLok kembali.\n", appUpdate.Latest)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Konfirmasi menjalankan installer")
	return cmd
}

func newUpdateRollbackCommand(jsonOut *bool) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Kembalikan binary aplikasi dari backup installer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return devlokerr.New(devlokerr.CodeCanceled, "Konfirmasi diperlukan untuk rollback.", "Tambahkan --yes setelah yakin.")
			}
			exe, err := os.Executable()
			if err != nil {
				return err
			}
			appDir := filepath.Dir(exe)
			backup := update.RollbackBackupDir(appDir)
			if _, err := os.Stat(filepath.Join(backup, "devlok-gui.exe")); err != nil {
				return devlokerr.New(devlokerr.CodeDependencyMissing, "Backup aplikasi sebelumnya tidak ditemukan.", "Rollback hanya tersedia setelah upgrade melalui installer DevLok.")
			}
			if err := stack.Down(""); err != nil {
				return err
			}
			if err := update.ScheduleRollback(appDir); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "update.rollback", true, map[string]string{"backup": backup})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rollback dijadwalkan dari:\n%s\nDevLok GUI akan ditutup, binary dipulihkan, lalu GUI dijalankan kembali.\n", backup)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Konfirmasi rollback")
	return cmd
}
