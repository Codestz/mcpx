package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/mcp"
	"github.com/fatih/color"
)

// outputMode controls how results are displayed.
type outputMode int

const (
	outputPretty outputMode = iota
	outputJSON
	outputQuiet
)

// output handles formatted output to stdout/stderr.
type output struct {
	mode   outputMode
	stdout io.Writer
	stderr io.Writer
}

func newOutput(mode outputMode) *output {
	return &output{
		mode:   mode,
		stdout: os.Stdout,
		stderr: os.Stderr,
	}
}

// printResult formats and prints a tool call result.
func (o *output) printResult(result *mcp.CallResult) error {
	if o.mode == outputQuiet {
		return nil
	}

	if o.mode == outputJSON {
		return o.printJSON(result)
	}

	// Pretty mode: print content blocks.
	for _, c := range result.Content {
		switch c.Type {
		case "text":
			fmt.Fprintln(o.stdout, c.Text)
		case "image":
			mime := c.MimeType
			if mime == "" {
				mime = "unknown"
			}
			fmt.Fprintf(o.stdout, "[image: %s, %d bytes base64]\n", mime, len(c.Data))
		case "audio":
			mime := c.MimeType
			if mime == "" {
				mime = "unknown"
			}
			fmt.Fprintf(o.stdout, "[audio: %s, %d bytes base64]\n", mime, len(c.Data))
		case "resource":
			if c.Resource != nil {
				if c.Resource.Text != "" {
					fmt.Fprintf(o.stdout, "[resource: %s]\n%s\n", c.Resource.URI, c.Resource.Text)
				} else {
					fmt.Fprintf(o.stdout, "[resource: %s, %d bytes base64]\n", c.Resource.URI, len(c.Resource.Blob))
				}
			} else {
				fmt.Fprintln(o.stdout, "[resource content]")
			}
		default:
			fmt.Fprintf(o.stdout, "[%s content]\n", c.Type)
		}
	}
	return nil
}

// printPing displays the result of a server health check.
func (o *output) printPing(serverName string, toolCount int, elapsed time.Duration) error {
	if o.mode == outputQuiet {
		return nil
	}

	ms := elapsed.Milliseconds()

	if o.mode == outputJSON {
		return o.printJSON(map[string]any{
			"server": serverName,
			"status": "ok",
			"tools":  toolCount,
			"ms":     ms,
		})
	}

	green := color.New(color.FgGreen)
	dim := color.New(color.FgHiBlack)

	fmt.Fprintf(o.stdout, "%s: ", serverName)
	green.Fprint(o.stdout, "ok")
	dim.Fprintf(o.stdout, " (%d tools, %dms)", toolCount, ms)
	fmt.Fprintln(o.stdout)

	return nil
}

// printJSON writes v as indented JSON to stdout.
func (o *output) printJSON(v any) error {
	enc := json.NewEncoder(o.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printTools displays a list of tools.
func (o *output) printTools(serverName string, tools []mcp.Tool) error {
	if o.mode == outputJSON {
		return o.printJSON(tools)
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	if len(tools) == 0 {
		fmt.Fprintf(o.stdout, "No tools found for server %q\n", serverName)
		return nil
	}

	bold.Fprintf(o.stdout, "%s", serverName)
	fmt.Fprintf(o.stdout, " (%d tools)\n\n", len(tools))

	for _, t := range tools {
		bold.Fprintf(o.stdout, "  %s", t.Name)
		if t.Description != "" {
			dim.Fprintf(o.stdout, "  %s", truncate(t.Description, 72))
		}
		fmt.Fprintln(o.stdout)
	}

	return nil
}

// printServers displays configured servers.
func (o *output) printServers(servers map[string]serverInfo) error {
	if o.mode == outputJSON {
		return o.printJSON(servers)
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	if len(servers) == 0 {
		fmt.Fprintln(o.stdout, "No servers configured.")
		fmt.Fprintln(o.stdout, "Add servers to .mcpx/config.yml or ~/.mcpx/config.yml")
		return nil
	}

	for name, info := range servers {
		bold.Fprintf(o.stdout, "  %s", name)
		dim.Fprintf(o.stdout, "  %s %s", info.Transport, info.Command)
		if info.Daemon {
			fmt.Fprintf(o.stdout, " (daemon)")
		}
		fmt.Fprintln(o.stdout)
	}

	return nil
}

// serverInfo is a summary of a configured server for display.
type serverInfo struct {
	Command   string `json:"command"`
	Transport string `json:"transport"`
	Daemon    bool   `json:"daemon"`
}

// printServerHelp displays a dynamic help page for a server, showing all tools.
func (o *output) printServerHelp(serverName string, sc *config.ServerConfig, tools []mcp.Tool) error {
	if o.mode == outputJSON {
		return o.printJSON(tools)
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	bold.Fprintf(o.stdout, "%s", serverName)
	dim.Fprintf(o.stdout, "  %s", sc.Command)
	if sc.Daemon {
		dim.Fprint(o.stdout, " (daemon)")
	}
	fmt.Fprintln(o.stdout)
	fmt.Fprintln(o.stdout)

	fmt.Fprintln(o.stdout, "Usage:")
	fmt.Fprintf(o.stdout, "  mcpx %s <tool> [flags]\n", serverName)
	fmt.Fprintf(o.stdout, "  mcpx %s <tool> --help       Show tool flags\n", serverName)
	fmt.Fprintf(o.stdout, "  mcpx %s <tool> --stdin      Read args from stdin JSON\n", serverName)
	fmt.Fprintln(o.stdout)

	fmt.Fprintf(o.stdout, "Available tools (%d):\n\n", len(tools))

	for _, t := range tools {
		bold.Fprintf(o.stdout, "  %s", t.Name)
		if t.Description != "" {
			dim.Fprintf(o.stdout, "  %s", truncate(t.Description, 60))
		}
		fmt.Fprintln(o.stdout)
	}

	fmt.Fprintln(o.stdout)
	dim.Fprintln(o.stdout, "Use --help on any tool to see its flags.")
	return nil
}

// printToolsVerbose displays tools with all their flags — full discovery in one call.
func (o *output) printToolsVerbose(serverName string, tools []mcp.Tool) error {
	if o.mode == outputJSON {
		return o.printJSON(tools)
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)

	bold.Fprintf(o.stdout, "%s", serverName)
	fmt.Fprintf(o.stdout, " (%d tools)\n", len(tools))

	for _, t := range tools {
		fmt.Fprintln(o.stdout)
		bold.Fprintf(o.stdout, "  %s", t.Name)
		fmt.Fprintln(o.stdout)

		if t.Description != "" {
			fmt.Fprintf(o.stdout, "    %s\n", truncate(t.Description, 76))
		}

		if len(t.InputSchema.Properties) == 0 {
			dim.Fprintln(o.stdout, "    (no flags)")
			continue
		}

		required := make(map[string]bool)
		for _, r := range t.InputSchema.Required {
			required[r] = true
		}

		propNames := make([]string, 0, len(t.InputSchema.Properties))
		for name := range t.InputSchema.Properties {
			propNames = append(propNames, name)
		}
		sort.Strings(propNames)

		for _, name := range propNames {
			prop := t.InputSchema.Properties[name]
			fmt.Fprintf(o.stdout, "    --%s", name)
			dim.Fprintf(o.stdout, " %s", flagTypeLabel(prop.Type))
			if required[name] {
				yellow.Fprint(o.stdout, " *")
			}
			fmt.Fprintln(o.stdout)
		}
	}

	return nil
}

// printDryRun shows what would be executed without executing it.
func (o *output) printDryRun(serverName, toolName string, command string, args []string, env map[string]string, toolArgs map[string]any) {
	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)

	bold.Fprintln(o.stdout, "Dry run — nothing will be executed")
	fmt.Fprintln(o.stdout)

	dim.Fprint(o.stdout, "Server:  ")
	fmt.Fprintln(o.stdout, serverName)

	dim.Fprint(o.stdout, "Tool:    ")
	fmt.Fprintln(o.stdout, toolName)

	dim.Fprint(o.stdout, "Command: ")
	fmt.Fprintf(o.stdout, "%s %s\n", command, strings.Join(args, " "))

	if len(env) > 0 {
		dim.Fprintln(o.stdout, "Env:")
		for k, v := range env {
			fmt.Fprintf(o.stdout, "  %s=%s\n", k, v)
		}
	}

	if len(toolArgs) > 0 {
		dim.Fprintln(o.stdout, "Arguments:")
		data, _ := json.MarshalIndent(toolArgs, "  ", "  ")
		fmt.Fprintf(o.stdout, "  %s\n", data)
	}
}

// printToolHelp displays detailed help for a single tool, showing all flags
// with their types, required status, and descriptions.
func (o *output) printToolHelp(serverName string, tool *mcp.Tool) {
	if o.mode == outputJSON {
		o.printJSON(tool)
		return
	}

	bold := color.New(color.Bold)
	dim := color.New(color.FgHiBlack)
	yellow := color.New(color.FgYellow)

	// Header.
	bold.Fprintf(o.stdout, "%s", tool.Name)
	dim.Fprintf(o.stdout, "  (%s)\n", serverName)

	if tool.Description != "" {
		fmt.Fprintf(o.stdout, "\n%s\n", tool.Description)
	}

	fmt.Fprintf(o.stdout, "\nUsage:\n")
	fmt.Fprintf(o.stdout, "  mcpx %s %s [flags]\n", serverName, tool.Name)

	if len(tool.InputSchema.Properties) == 0 {
		fmt.Fprintf(o.stdout, "\nNo flags.\n")
		return
	}

	// Build required set for quick lookup.
	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}

	// Sort properties.
	propNames := make([]string, 0, len(tool.InputSchema.Properties))
	for name := range tool.InputSchema.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	// Calculate column widths.
	maxFlag := 0
	maxType := 0
	for _, name := range propNames {
		if len(name)+2 > maxFlag { // +2 for "--"
			maxFlag = len(name) + 2
		}
		prop := tool.InputSchema.Properties[name]
		typeStr := flagTypeLabel(prop.Type)
		if len(typeStr) > maxType {
			maxType = len(typeStr)
		}
	}

	fmt.Fprintf(o.stdout, "\nFlags:\n")
	for _, name := range propNames {
		prop := tool.InputSchema.Properties[name]

		// Flag name.
		flagStr := fmt.Sprintf("--%s", name)
		fmt.Fprintf(o.stdout, "  %-*s", maxFlag+2, flagStr)

		// Type.
		typeStr := flagTypeLabel(prop.Type)
		dim.Fprintf(o.stdout, " %-*s", maxType+1, typeStr)

		// Required marker.
		if required[name] {
			yellow.Fprintf(o.stdout, " (required)")
		} else {
			fmt.Fprintf(o.stdout, "           ")
		}

		// Description.
		if prop.Description != "" {
			fmt.Fprintf(o.stdout, "  %s", prop.Description)
		}

		// Default value.
		if prop.Default != nil {
			dim.Fprintf(o.stdout, " (default: %v)", prop.Default)
		}

		// Enum values.
		if len(prop.Enum) > 0 {
			vals := make([]string, len(prop.Enum))
			for i, v := range prop.Enum {
				vals[i] = fmt.Sprintf("%v", v)
			}
			dim.Fprintf(o.stdout, " [%s]", strings.Join(vals, ", "))
		}

		fmt.Fprintln(o.stdout)
	}
}

func flagTypeLabel(jsonType string) string {
	switch jsonType {
	case "string":
		return "string"
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	case "array":
		return "json[]"
	case "object":
		return "json{}"
	default:
		return jsonType
	}
}

// errorf prints a formatted error to stderr.
func (o *output) errorf(format string, args ...any) {
	red := color.New(color.FgRed)
	red.Fprintf(o.stderr, "error: "+format+"\n", args...)
}

func truncate(s string, max int) string {
	// Truncate at first newline.
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
