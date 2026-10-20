package cli

import (
	"encoding/json"
	"io"
)

func writeJSON(w io.Writer, command string, ok bool, data any) error {
	payload := struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Data    any    `json:"data,omitempty"`
	}{
		OK:      ok,
		Command: command,
		Data:    data,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
