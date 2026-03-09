package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPrintPing(t *testing.T) {
	tests := []struct {
		name       string
		mode       outputMode
		server     string
		toolCount  int
		elapsed    time.Duration
		wantOutput string
		wantJSON   map[string]any
		wantEmpty  bool
	}{
		{
			name:       "pretty output",
			mode:       outputPretty,
			server:     "serena",
			toolCount:  21,
			elapsed:    47 * time.Millisecond,
			wantOutput: "serena: ok (21 tools, 47ms)",
		},
		{
			name:      "json output",
			mode:      outputJSON,
			server:    "serena",
			toolCount: 21,
			elapsed:   47 * time.Millisecond,
			wantJSON: map[string]any{
				"server": "serena",
				"status": "ok",
				"tools":  float64(21),
				"ms":     float64(47),
			},
		},
		{
			name:      "quiet output",
			mode:      outputQuiet,
			server:    "serena",
			toolCount: 21,
			elapsed:   47 * time.Millisecond,
			wantEmpty: true,
		},
		{
			name:       "zero tools",
			mode:       outputPretty,
			server:     "empty-server",
			toolCount:  0,
			elapsed:    5 * time.Millisecond,
			wantOutput: "empty-server: ok (0 tools, 5ms)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			out := &output{mode: tt.mode, stdout: &buf, stderr: &buf}

			err := out.printPing(tt.server, tt.toolCount, tt.elapsed)
			if err != nil {
				t.Fatalf("printPing() error: %v", err)
			}

			got := buf.String()

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty output, got %q", got)
				}
				return
			}

			if tt.wantJSON != nil {
				var parsed map[string]any
				if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
					t.Fatalf("invalid JSON output: %v\nraw: %s", err, got)
				}
				for k, want := range tt.wantJSON {
					if parsed[k] != want {
						t.Errorf("JSON key %q: got %v, want %v", k, parsed[k], want)
					}
				}
				return
			}

			if !strings.Contains(got, tt.wantOutput) {
				t.Errorf("output %q does not contain %q", got, tt.wantOutput)
			}
		})
	}
}
