package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"devlok/internal/instance"
	"devlok/internal/platform"
	"devlok/internal/stack"
	"devlok/internal/supervisor"
)

func newSuperviseCommand() *cobra.Command {
	var stateDir string
	var once bool
	var interval time.Duration
	cmd := &cobra.Command{
		Use:    "supervise",
		Short:  "Jalankan supervisor latar untuk stack aktif",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			state := stateDir
			if state == "" {
				var err error
				state, err = platform.Current().StateDir()
				if err != nil {
					return err
				}
			}
			lock, err := instance.Acquire("supervisor:" + state)
			if errors.Is(err, instance.ErrAlreadyRunning) {
				return nil
			}
			if err != nil {
				return err
			}
			defer lock.Close()
			return superviseLoop(state, interval, once)
		},
	}
	cmd.Flags().StringVar(&stateDir, "state", "", "State dir DevLok")
	cmd.Flags().BoolVar(&once, "once", false, "Jalankan satu siklus lalu keluar")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Interval pemeriksaan")
	return cmd
}

func superviseLoop(state string, interval time.Duration, once bool) error {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	stackPath := filepath.Join(state, "state", "stack.json")
	servicesPath := filepath.Join(state, "state", "services.json")
	mgr, err := supervisor.NewManager(servicesPath)
	if err != nil {
		return err
	}
	failures := map[string]int{}
	nextAttempt := map[string]time.Time{}
	unhealthy := map[string]int{}
	stopPath := filepath.Join(state, "state", "supervisor.stop")
	for {
		if _, err := os.Stat(stopPath); err == nil {
			return nil
		}
		if _, err := os.Stat(stackPath); os.IsNotExist(err) {
			return nil
		}
		if stack.SupervisorPaused(state) {
			if once {
				return nil
			}
			time.Sleep(interval)
			continue
		}
		specs, err := stack.LoadSupervisorSpecs(state)
		if err != nil {
			supervisorLog(state, "config: "+err.Error())
			if once {
				return err
			}
			time.Sleep(interval)
			continue
		}
		if len(specs) == 0 {
			return nil
		}
		mgr.Reconcile()
		for _, spec := range specs {
			status, ok := mgr.Get(spec.Name)
			if !ok || status.State == supervisor.StateStopped || status.State == supervisor.StateFailed {
				if time.Now().Before(nextAttempt[spec.Name]) {
					continue
				}
				if _, err := mgr.Start(spec); err != nil {
					failures[spec.Name]++
					nextAttempt[spec.Name] = time.Now().Add(restartBackoff(failures[spec.Name]))
					delete(unhealthy, spec.Name)
					supervisorLog(state, "start "+spec.Name+" gagal: "+err.Error())
				} else {
					delete(failures, spec.Name)
					delete(nextAttempt, spec.Name)
					supervisorLog(state, "start "+spec.Name)
				}
				continue
			}
			if status.State != supervisor.StateRunning {
				continue
			}
			if spec.Health.TCPPort == 0 && spec.Health.HTTPURL == "" {
				continue
			}
			if supervisor.HealthOK(spec.Health) {
				unhealthy[spec.Name] = 0
				delete(failures, spec.Name)
				continue
			}
			unhealthy[spec.Name]++
			if unhealthy[spec.Name] < 3 {
				continue
			}
			unhealthy[spec.Name] = 0
			if _, err := mgr.Restart(spec); err != nil {
				failures[spec.Name]++
				nextAttempt[spec.Name] = time.Now().Add(restartBackoff(failures[spec.Name]))
				supervisorLog(state, "restart "+spec.Name+" gagal: "+err.Error())
			} else {
				delete(failures, spec.Name)
				delete(nextAttempt, spec.Name)
				supervisorLog(state, "restart "+spec.Name+" karena health check gagal")
			}
		}
		if once {
			return nil
		}
		time.Sleep(interval)
	}
}

func restartBackoff(failures int) time.Duration {
	d := 2 * time.Second
	for i := 1; i < failures && d < 30*time.Second; i++ {
		d *= 2
	}
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func supervisorLog(state, message string) {
	dir := filepath.Join(state, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "supervisor.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), message)
}
