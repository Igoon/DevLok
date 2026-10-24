package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type WildcardStatus struct {
	TLD       string `json:"tld"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid,omitempty"`
	Resolver  bool   `json:"resolver"`
}

func newWildcardCommand(jsonOut *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wildcard",
		Short: "Domain wildcard (mis. *.test) via dnsmasq [macOS]",
	}
	var onTLD, offTLD string
	on := &cobra.Command{
		Use:   "on",
		Short: "Aktifkan wildcard untuk sebuah TLD",
		RunE: func(c *cobra.Command, args []string) error {
			msg, err := wildcardEnable(onTLD)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(c.OutOrStdout(), "wildcard.on", true, map[string]string{"tld": onTLD})
			}
			fmt.Fprintln(c.OutOrStdout(), msg)
			return nil
		},
	}
	on.Flags().StringVar(&onTLD, "tld", "test", "TLD, mis. test")
	off := &cobra.Command{
		Use:   "off",
		Short: "Matikan wildcard dan hapus resolver",
		RunE: func(c *cobra.Command, args []string) error {
			msg, err := wildcardDisable(offTLD)
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(c.OutOrStdout(), "wildcard.off", true, map[string]string{"tld": offTLD})
			}
			fmt.Fprintln(c.OutOrStdout(), msg)
			return nil
		},
	}
	off.Flags().StringVar(&offTLD, "tld", "test", "TLD, mis. test")
	status := &cobra.Command{
		Use:   "status",
		Short: "Status wildcard dnsmasq",
		RunE: func(c *cobra.Command, args []string) error {
			st, err := wildcardStatus()
			if err != nil {
				return err
			}
			if *jsonOut {
				return writeJSON(c.OutOrStdout(), "wildcard.status", true, st)
			}
			fmt.Fprintf(c.OutOrStdout(), "dnsmasq terpasang: %v\n", st.Installed)
			fmt.Fprintf(c.OutOrStdout(), "wildcard jalan: %v (PID %d)\n", st.Running, st.PID)
			fmt.Fprintf(c.OutOrStdout(), "resolver /etc/resolver/%s: %v\n", st.TLD, st.Resolver)
			return nil
		},
	}
	cmd.AddCommand(on, off, status)
	return cmd
}
