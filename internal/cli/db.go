package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/config"
	dbpkg "devlok/internal/db"
	"devlok/internal/devlokerr"
	"devlok/internal/platform"
	rt "devlok/internal/runtime"
)

func newDBCommand(jsonOut *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Kelola database MySQL per project",
	}
	cmd.AddCommand(newDBListCommand(jsonOut))
	cmd.AddCommand(newDBCreateCommand(jsonOut))
	cmd.AddCommand(newDBDropCommand(jsonOut))
	cmd.AddCommand(newDBResetCommand(jsonOut))
	cmd.AddCommand(newDBBackupCommand(jsonOut))
	cmd.AddCommand(newDBRestoreCommand(jsonOut))
	cmd.AddCommand(newDBSeedCommand(jsonOut))
	cmd.AddCommand(newDBShellCommand())
	cmd.AddCommand(newDBCredsCommand(jsonOut))
	cmd.AddCommand(newDBAdminCommand())
	return cmd
}

func newDBResetCommand(jsonOut *bool) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Pindahkan datadir MySQL dan inisialisasi ulang",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return devlokerr.New(devlokerr.CodeCanceled, "Konfirmasi diperlukan untuk reset.", "Tambahkan --yes. Datadir lama dipindahkan ke folder .bak, bukan dihapus.")
			}
			mgr, err := newDBManager("", nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()
			backup, err := mgr.Reset(ctx)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.reset", true, map[string]string{"backup": backup})
			}
			if backup != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Data MySQL dipindahkan ke:\n%s\nJalankan dl up untuk inisialisasi ulang.\n", backup)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Tidak ada datadir. Jalankan dl up untuk inisialisasi.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Konfirmasi reset")
	return cmd
}

func newDBListCommand(jsonOut *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Daftar database project",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newDBManager("", nil)
			if err != nil {
				return err
			}
			dbs, err := mgr.Databases()
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.list", true, map[string]any{"databases": dbs, "port": mgr.Port})
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(tw, "DATABASE\tUSER\tPROJECT")
			for _, dc := range dbs {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", dc.Name, dc.User, dc.Project)
			}
			return tw.Flush()
		},
	}
}

func newDBCreateCommand(jsonOut *bool) *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Buat database dan user project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, dbName, err := dbTarget(args)
			if err != nil {
				return err
			}
			mgr, err := newDBManager(version, runtimeProgress(cmd, *jsonOut))
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Minute)
			defer cancel()
			dc, err := mgr.EnsureDatabase(ctx, project, dbName)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.create", true, map[string]any{"database": dc.Name, "user": dc.User, "port": mgr.Port})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Database: %s\nUser: %s\nPort: %d\n", dc.Name, dc.User, mgr.Port)
			fmt.Fprintf(cmd.OutOrStdout(), "Lihat password: dl db creds %s\n", dc.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "mysql", "", "Versi MySQL (default 8.4)")
	return cmd
}

func newDBDropCommand(jsonOut *bool) *cobra.Command {
	var yes bool
	var version string
	cmd := &cobra.Command{
		Use:   "drop <name>",
		Short: "Hapus database dan user project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return devlokerr.New(devlokerr.CodeCanceled, "Konfirmasi diperlukan untuk drop.", "Tambahkan --yes setelah yakin.")
			}
			mgr, err := newDBManager(version, nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()
			if err := mgr.Drop(ctx, args[0]); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.drop", true, map[string]string{"dropped": args[0]})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Database dihapus: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Konfirmasi hapus")
	cmd.Flags().StringVar(&version, "mysql", "", "Versi MySQL")
	return cmd
}

func newDBBackupCommand(jsonOut *bool) *cobra.Command {
	var out string
	var version string
	cmd := &cobra.Command{
		Use:   "backup [name]",
		Short: "Backup database ke file .sql.gz",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, dbName, err := dbTarget(args)
			if err != nil {
				return err
			}
			mgr, err := newDBManager(version, nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer cancel()
			path, err := mgr.Backup(ctx, dbName, out)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.backup", true, map[string]string{"file": path})
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&out, "output", "o", "", "Path file backup")
	cmd.Flags().StringVar(&version, "mysql", "", "Versi MySQL")
	return cmd
}

func newDBRestoreCommand(jsonOut *bool) *cobra.Command {
	var file string
	var yes bool
	var version string
	cmd := &cobra.Command{
		Use:   "restore <name>",
		Short: "Restore database dari file SQL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return devlokerr.New(devlokerr.CodeCanceled, "Konfirmasi diperlukan untuk restore.", "Tambahkan --yes setelah yakin.")
			}
			if file == "" {
				return devlokerr.New(devlokerr.CodeInvalidArgument, "File restore wajib diisi.", "Gunakan -f file.sql atau -f file.sql.gz.")
			}
			mgr, err := newDBManager(version, nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer cancel()
			if err := mgr.Restore(ctx, args[0], file); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.restore", true, map[string]string{"restored": args[0], "file": file})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restore selesai: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "File SQL atau SQL.gz")
	cmd.Flags().BoolVar(&yes, "yes", false, "Konfirmasi restore")
	cmd.Flags().StringVar(&version, "mysql", "", "Versi MySQL")
	return cmd
}

func newDBSeedCommand(jsonOut *bool) *cobra.Command {
	var version string
	cmd := &cobra.Command{
		Use:   "seed [name]",
		Short: "Jalankan db/seed.sql",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest, projectDir, err := currentManifest()
			if err != nil {
				return err
			}
			dbName := manifest.DB.Name
			if len(args) == 1 {
				dbName = args[0]
			}
			seed := manifest.DB.Seed
			if seed == "" {
				seed = filepath.Join("db", "seed.sql")
			}
			seedPath := seed
			if !filepath.IsAbs(seedPath) {
				seedPath = filepath.Join(projectDir, seed)
			}
			if _, err := os.Stat(seedPath); err != nil {
				return devlokerr.New(devlokerr.CodeInvalidArgument, "File seed tidak ditemukan: "+seedPath, "Isi db.seed di manifest atau buat db/seed.sql.")
			}
			mgr, err := newDBManager(version, nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Minute)
			defer cancel()
			if err := mgr.Seed(ctx, dbName, seedPath); err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.seed", true, map[string]string{"database": dbName, "file": seedPath})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Seed selesai: %s\n", dbName)
			return nil
		},
	}
	cmd.Flags().StringVar(&version, "mysql", "", "Versi MySQL")
	return cmd
}

func newDBShellCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "shell [name]",
		Short: "Buka mysql client interaktif",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			mgr, err := newDBManager("", nil)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Hour)
			defer cancel()
			return mgr.Shell(ctx, name)
		},
	}
}

func newDBCredsCommand(jsonOut *bool) *cobra.Command {
	var root bool
	cmd := &cobra.Command{
		Use:   "creds [name]",
		Short: "Tampilkan kredensial database lokal",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := newDBManager("", nil)
			if err != nil {
				return err
			}
			if root {
				pw, err := mgr.RootPassword()
				if err != nil {
					return err
				}
				if pw == "" {
					return devlokerr.New(devlokerr.CodeStateInconsistent, "Root password belum dibuat.", "Jalankan dl up atau dl db create.")
				}
				if *jsonOut {
					return writeJSON(cmd.OutOrStdout(), "db.creds", true, map[string]any{"user": "root", "password": pw, "port": mgr.Port})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "user=root\npassword=%s\nport=%d\n", pw, mgr.Port)
				return nil
			}
			_, dbName, err := dbTarget(args)
			if err != nil {
				return err
			}
			dc, err := mgr.Database(dbName)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(cmd.OutOrStdout(), "db.creds", true, map[string]any{"database": dc.Name, "user": dc.User, "password": dc.Password, "port": mgr.Port})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "database=%s\nuser=%s\npassword=%s\nport=%d\n", dc.Name, dc.User, dc.Password, mgr.Port)
			return nil
		},
	}
	cmd.Flags().BoolVar(&root, "root", false, "Tampilkan kredensial root")
	return cmd
}

func newDBAdminCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "admin",
		Short: "URL dan kredensial phpMyAdmin",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "phpMyAdmin: https://phpmyadmin.test")
			fmt.Fprintln(cmd.OutOrStdout(), "Root password: dl db creds --root")
			fmt.Fprintln(cmd.OutOrStdout(), "Project credentials: dl db creds <database>")
			return nil
		},
	}
}

func newDBManager(version string, progress func(rt.Progress)) (*dbpkg.Manager, error) {
	state, err := platform.Current().StateDir()
	if err != nil {
		return nil, err
	}
	port := 0
	if m, project, err := currentManifest(); err == nil {
		port = m.Ports.MySQL
		if version == "" {
			version = m.Services["mysql"]
			if lock, err := config.LoadLock(filepath.Join(project, config.LockFileName)); err == nil {
				if v := lock.Services["mysql"].Version; v != "" {
					version = v
				}
			}
		}
	}
	return dbpkg.New(dbpkg.Options{StateDir: state, Version: version, Port: port, Progress: progress})
}

func dbTarget(args []string) (project, dbName string, err error) {
	m, _, mErr := currentManifest()
	if len(args) == 1 {
		dbName = args[0]
		if mErr == nil {
			project = m.Name
		} else {
			project = args[0]
		}
		return project, dbName, nil
	}
	if mErr != nil {
		return "", "", devlokerr.New(devlokerr.CodeManifestNotFound, "Manifest tidak ditemukan.", "Tentukan nama database atau jalankan dl init.")
	}
	return m.Name, m.DB.Name, nil
}

func currentManifest() (*config.Manifest, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	path, ok := config.Find(cwd)
	if !ok {
		return nil, "", devlokerr.New(devlokerr.CodeManifestNotFound, "Manifest tidak ditemukan.", "Jalankan dl init atau dl site create.")
	}
	m, err := config.Load(path)
	if err != nil {
		return nil, "", err
	}
	return m, filepath.Dir(path), nil
}
