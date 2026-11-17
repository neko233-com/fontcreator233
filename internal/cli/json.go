package cli

import (
	"encoding/json"
	"fmt"
)

func (a *app) writeJSON(v any) {
	enc := json.NewEncoder(a.io.Out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(a.io.Err, "fontcreator233: encode JSON: %v\n", err)
	}
}

func okMap(command string, extra map[string]any) map[string]any {
	m := map[string]any{"ok": true, "command": command, "version": version}
	for k, v := range extra {
		m[k] = v
	}
	return m
}
