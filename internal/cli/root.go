package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"devlok/internal/devlokerr"
	"devlok/internal/version"
)

type exitError struct {
	code int
}

func (e exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

func Execute() int {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			return ee.code
		}
		printError(root, err)
		return devlokerr.ExitCode(err)
	}
	return 0
}

func newRootCommand() *cobra.Command {
	var jsonOut, verbose bool
	root := &cobra.Command{
		Use:           "dl",
		Short:         "Local development environment untuk Windows",
		Long:          "DevLok mengelola Nginx, PHP, MySQL, dan tool pendukung dalam satu CLI dan GUI.",
		Version:       version.Current().Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output JSON")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "Log detail")
	root.AddCommand(newDBCommand(&jsonOut))
	root.AddCommand(newExportCommand(&jsonOut))
	root.AddCommand(newImportCommand(&jsonOut))
	root.AddCommand(newLockCommand(&jsonOut))
	root.AddCommand(newDoctorCommand(&jsonOut))
	root.AddCommand(newDownCommand(&jsonOut))
	root.AddCommand(newEditCommand())
	root.AddCommand(newSiteCommand(&jsonOut))
	root.AddCommand(newStatusCommand(&jsonOut))
	root.AddCommand(newUpCommand(&jsonOut))
	root.AddCommand(newInitCommand(&jsonOut))
	root.AddCommand(newPortsCommand(&jsonOut))
	root.AddCommand(newRunCommand())
	root.AddCommand(newShellCommand())
	root.AddCommand(newRuntimeCommand(&jsonOut))
	root.AddCommand(newUpdateCommand(&jsonOut))
	root.AddCommand(newUICommand())
	root.AddCommand(newSuperviseCommand())
	root.AddCommand(newVersionCommand(&jsonOut))
	return root
}

func printError(root *cobra.Command, err error) {
	jsonOut, _ := root.PersistentFlags().GetBool("json")
	e, ok := devlokerr.As(err)
	if jsonOut {
		payload := map[string]any{"ok": false}
		if ok {
			payload["error"] = map[string]any{
				"code":    e.Code,
				"message": e.Message,
				"hint":    e.Hint,
				"details": e.Details,
			}
		} else {
			payload["error"] = map[string]any{"code": "INTERNAL", "message": err.Error()}
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Fprintln(os.Stderr, string(data))
		return
	}
	if ok {
		fmt.Fprintf(os.Stderr, "dl: %s: %s\n", e.Code, e.Message)
		if e.Hint != "" {
			fmt.Fprintf(os.Stderr, "hint: %s\n", e.Hint)
		}
		if e.Err != nil {
			fmt.Fprintf(os.Stderr, "detail: %v\n", e.Err)
		}
		return
	}
	fmt.Fprintf(os.Stderr, "dl: %v\n", err)
}
