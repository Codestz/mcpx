package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/codestz/mcpx/internal/config"
	"github.com/codestz/mcpx/internal/daemon"
	"github.com/codestz/mcpx/internal/lifecycle"
	"github.com/codestz/mcpx/internal/mcp"
	"github.com/codestz/mcpx/internal/resolver"
	"github.com/codestz/mcpx/internal/secret"
	"github.com/codestz/mcpx/internal/security"
	"github.com/spf13/cobra"
)

// buildServerCommand creates a Cobra command for a configured MCP server.
// It uses DisableFlagParsing so that tool flags (--file_mask, etc.) pass through
// to the dynamic tool parser instead of being rejected by Cobra.
func buildServerCommand(name string, sc *config.ServerConfig, globalSec *config.SecurityConfig, opts *globalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:                name,
		Short:              fmt.Sprintf("MCP server: %s", name),
		DisableFlagParsing: true,
		ValidArgsFunction:  completeToolNames(name, sc),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Extract global flags from args since flag parsing is disabled.
			// --help is only intercepted at the server level (no tool name yet).
			// If a tool name is present, --help passes through to per-tool help.
			var filtered []string
			hasHelp := false
			for i := 0; i < len(args); i++ {
				switch args[i] {
				case "--json":
					opts.jsonOutput = true
				case "--quiet":
					opts.quiet = true
				case "--dry-run":
					opts.dryRun = true
				case "--pick":
					if i+1 < len(args) {
						i++
						opts.pick = args[i]
					}
				case "--timeout":
					if i+1 < len(args) {
						i++
						opts.timeout = args[i]
					}
				case "--help", "-h":
					hasHelp = true
				default:
					filtered = append(filtered, args[i])
				}
			}
			args = filtered

			// No tool name: show dynamic server help (connects to server, lists tools).
			if len(args) == 0 {
				return showServerHelp(cmd.Context(), name, sc, opts)
			}

			// Re-inject --help for per-tool help if a tool name is present.
			if hasHelp {
				args = append(args, "--help")
			}

			// Handle "list" subcommand.
			if args[0] == "list" {
				out := newOutput(opts.outputMode())
				client, cleanup, err := connectServer(cmd.Context(), name, sc)
				if err != nil {
					return err
				}
				defer cleanup()

				tools, err := client.ListTools(cmd.Context())
				if err != nil {
					return fmt.Errorf("list tools: %w", err)
				}

				return out.printTools(name, tools)
			}

			// Handle "info" subcommand.
			if args[0] == "info" {
				return handleServerInfo(cmd.Context(), name, sc, opts)
			}

			// Handle "prompt" subcommand.
			if args[0] == "prompt" {
				return handlePrompt(cmd.Context(), name, sc, args[1:], opts)
			}

			// Handle "resource" subcommand.
			if args[0] == "resource" {
				return handleResource(cmd.Context(), name, sc, args[1:], opts)
			}

			// Handle "generate" subcommand.
			if args[0] == "generate" {
				global := false
				format := "default"
				for _, a := range args[1:] {
					if a == "--global" {
						global = true
					}
					if strings.HasPrefix(a, "--format=") {
						format = strings.TrimPrefix(a, "--format=")
					}
					if a == "--format" {
						// Next arg is the value — handled below.
					}
				}
				// Handle --format <value> (space-separated).
				for i, a := range args[1:] {
					if a == "--format" && i+1 < len(args[1:]) {
						format = args[1:][i+1]
					}
				}
				return runGenerate(cmd.Context(), name, sc, global, format)
			}

			// Dynamic tool dispatch: mcpx <server> <tool> --flags
			toolName := args[0]
			toolArgs := args[1:]
			return runTool(cmd.Context(), name, sc, globalSec, toolName, toolArgs, opts)
		},
	}

	return cmd
}

// showServerHelp connects to a server and displays a dynamic help page
// listing all available tools, prompts, and resources.
func showServerHelp(ctx context.Context, serverName string, sc *config.ServerConfig, opts *globalOpts) error {
	out := newOutput(opts.outputMode())

	client, cleanup, err := connectServer(ctx, serverName, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	caps := client.ServerCapabilities()

	var prompts []mcp.Prompt
	if caps.Prompts != nil {
		prompts, _ = client.ListPrompts(ctx)
	}

	var resources []mcp.Resource
	if caps.Resources != nil {
		resources, _ = client.ListResources(ctx)
	}

	return out.printServerHelpFull(serverName, sc, tools, prompts, resources)
}

// runTool connects to a server, finds the named tool, parses flags, and executes.
func runTool(ctx context.Context, serverName string, sc *config.ServerConfig, globalSec *config.SecurityConfig, toolName string, rawArgs []string, opts *globalOpts) error {
	out := newOutput(opts.outputMode())

	// Check for help or stdin mode.
	wantHelp := false
	useStdin := false
	var filteredArgs []string
	for _, a := range rawArgs {
		switch a {
		case "--help", "-h":
			wantHelp = true
		case "--stdin":
			useStdin = true
		default:
			filteredArgs = append(filteredArgs, a)
		}
	}

	client, cleanup, err := connectServer(ctx, serverName, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	tools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	var tool *mcp.Tool
	for i := range tools {
		if tools[i].Name == toolName {
			tool = &tools[i]
			break
		}
	}
	if tool == nil {
		var names []string
		for _, t := range tools {
			names = append(names, t.Name)
		}
		return fmt.Errorf("tool %q not found in server %q\nAvailable tools: %s\nRun: mcpx %s --help",
			toolName, serverName, strings.Join(names, ", "), serverName)
	}

	if wantHelp {
		out.printToolHelp(serverName, tool)
		return nil
	}

	// Parse arguments: either from stdin JSON (with optional flag merge) or from flags.
	var toolArgs map[string]any
	if useStdin {
		toolArgs, err = parseStdinJSON()
		if err != nil {
			return fmt.Errorf("--stdin: %w", err)
		}
		// Merge CLI flags on top (flags win).
		if len(filteredArgs) > 0 {
			flagArgs, flagErr := parseToolFlagsPartial(tool, filteredArgs)
			if flagErr != nil {
				return enhanceParseError(flagErr, serverName, tool)
			}
			for k, v := range flagArgs {
				toolArgs[k] = v
			}
		}
		// Validate required fields against merged result.
		for _, req := range tool.InputSchema.Required {
			if _, ok := toolArgs[req]; !ok {
				return fmt.Errorf("required field %q not provided (via --stdin or flags)", req)
			}
		}
	} else {
		toolArgs, err = parseToolFlags(tool, filteredArgs)
		if err != nil {
			return enhanceParseError(err, serverName, tool)
		}
	}

	if opts.dryRun {
		resolvedArgs, resolvedEnv, err := resolveServerConfig(sc)
		if err != nil {
			return err
		}
		out.printDryRun(serverName, toolName, sc.Command, resolvedArgs, resolvedEnv, toolArgs)
		return nil
	}

	// Apply per-call timeout if specified.
	callCtx := ctx
	if opts.timeout != "" {
		d, err := time.ParseDuration(opts.timeout)
		if err != nil {
			return fmt.Errorf("--timeout: %w", err)
		}
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}

	// Evaluate security policies before calling the tool.
	// Use workspace security if cwd is inside a workspace.
	if globalSec != nil && globalSec.Enabled || sc.Security != nil {
		serverSec := sc.Security
		projectRoot, _ := findProjectRoot()
		if ws := config.ResolveWorkspace(sc, projectRoot); ws != nil && ws.Security != nil {
			serverSec = ws.Security
		}
		eval := security.NewEvaluator(serverName, globalSec, serverSec)
		secResult := eval.Evaluate(toolName, toolArgs)

		switch secResult.Action {
		case security.ActionDeny:
			// Log denial to audit if configured.
			logAudit(globalSec, serverName, toolName, toolArgs, secResult)
			msg := fmt.Sprintf("server %q: policy %q denied tool %q\n  Reason: %s",
				serverName, secResult.PolicyName, toolName, secResult.Message)
			if secResult.Details != "" {
				msg += fmt.Sprintf("\n  %s", secResult.Details)
			}
			return fmt.Errorf("%s", msg)
		case security.ActionWarn:
			fmt.Fprintf(os.Stderr, "mcpx: warning: %s\n", secResult.Message)
			logAudit(globalSec, serverName, toolName, toolArgs, secResult)
		default:
			logAudit(globalSec, serverName, toolName, toolArgs, secResult)
		}
	}

	result, err := client.CallTool(callCtx, toolName, toolArgs)
	if err != nil {
		return err
	}

	// Extract a specific field if --pick was specified.
	if opts.pick != "" {
		val, err := pickField(result, opts.pick)
		if err != nil {
			return fmt.Errorf("--pick %s: %w", opts.pick, err)
		}
		fmt.Fprintln(os.Stdout, val)
		return nil
	}

	return out.printResult(result)
}

// handleServerInfo connects to the server and displays its metadata and capabilities.
func handleServerInfo(ctx context.Context, name string, sc *config.ServerConfig, opts *globalOpts) error {
	out := newOutput(opts.outputMode())

	client, cleanup, err := connectServer(ctx, name, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	info := client.ServerInfo()
	caps := client.ServerCapabilities()
	version := client.ProtocolVersion()

	return out.printServerInfo(name, info, caps, version)
}

// handlePrompt dispatches prompt subcommands: list, <name> --help, <name> --args.
func handlePrompt(ctx context.Context, name string, sc *config.ServerConfig, args []string, opts *globalOpts) error {
	out := newOutput(opts.outputMode())

	if len(args) == 0 || args[0] == "list" {
		client, cleanup, err := connectServer(ctx, name, sc)
		if err != nil {
			return err
		}
		defer cleanup()

		prompts, err := client.ListPrompts(ctx)
		if err != nil {
			return fmt.Errorf("list prompts: %w", err)
		}
		return out.printPrompts(name, prompts)
	}

	promptName := args[0]
	promptArgs := args[1:]

	// Check for --help.
	wantHelp := false
	var filteredArgs []string
	for _, a := range promptArgs {
		if a == "--help" || a == "-h" {
			wantHelp = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}

	client, cleanup, err := connectServer(ctx, name, sc)
	if err != nil {
		return err
	}
	defer cleanup()

	if wantHelp {
		prompts, err := client.ListPrompts(ctx)
		if err != nil {
			return fmt.Errorf("list prompts: %w", err)
		}
		for i := range prompts {
			if prompts[i].Name == promptName {
				out.printPromptHelp(name, &prompts[i])
				return nil
			}
		}
		return fmt.Errorf("prompt %q not found in server %q", promptName, name)
	}

	// Parse --key value pairs into map[string]string.
	promptMap := make(map[string]string)
	for i := 0; i < len(filteredArgs); i++ {
		arg := filteredArgs[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(filteredArgs) {
				i++
				promptMap[key] = filteredArgs[i]
			} else {
				return fmt.Errorf("flag --%s requires a value", key)
			}
		}
	}

	// Validate required arguments.
	prompts, err := client.ListPrompts(ctx)
	if err != nil {
		return fmt.Errorf("list prompts: %w", err)
	}
	for i := range prompts {
		if prompts[i].Name == promptName {
			for _, arg := range prompts[i].Arguments {
				if arg.Required {
					if _, ok := promptMap[arg.Name]; !ok {
						return fmt.Errorf("required argument --%s not provided\nRun: mcpx %s prompt %s --help", arg.Name, name, promptName)
					}
				}
			}
			break
		}
	}

	result, err := client.GetPrompt(ctx, promptName, promptMap)
	if err != nil {
		return err
	}

	return out.printPromptResult(result)
}

// handleResource dispatches resource subcommands: list, read <uri>.
func handleResource(ctx context.Context, name string, sc *config.ServerConfig, args []string, opts *globalOpts) error {
	out := newOutput(opts.outputMode())

	if len(args) == 0 || args[0] == "list" {
		client, cleanup, err := connectServer(ctx, name, sc)
		if err != nil {
			return err
		}
		defer cleanup()

		resources, err := client.ListResources(ctx)
		if err != nil {
			return fmt.Errorf("list resources: %w", err)
		}
		if err := out.printResources(name, resources); err != nil {
			return err
		}

		// Also list resource templates if available.
		templates, _ := client.ListResourceTemplates(ctx)
		if len(templates) > 0 {
			fmt.Fprintln(out.stdout)
			return out.printResourceTemplates(name, templates)
		}
		return nil
	}

	if args[0] == "read" {
		if len(args) < 2 {
			return fmt.Errorf("usage: mcpx %s resource read <uri>", name)
		}
		uri := args[1]

		client, cleanup, err := connectServer(ctx, name, sc)
		if err != nil {
			return err
		}
		defer cleanup()

		result, err := client.ReadResource(ctx, uri)
		if err != nil {
			return err
		}

		return out.printResourceResult(result)
	}

	return fmt.Errorf("unknown resource subcommand %q\nUsage: mcpx %s resource [list|read <uri>]", args[0], name)
}

// resolveStringValue resolves @file / @- / - syntax for flag values.
// "-" or "@-" reads from stdin, "@<path>" reads a file, bare "@" is literal.
func resolveStringValue(val, flagName string) (string, error) {
	if val == "-" || val == "@-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin for --%s: %w", flagName, err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	if len(val) > 1 && val[0] == '@' {
		path := val[1:]
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read file for --%s: %w", flagName, err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	return val, nil
}

// parseStdinJSON reads a JSON object from stdin and returns it as tool arguments.
func parseStdinJSON() (map[string]any, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("read stdin: %w", err)
	}
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return nil, fmt.Errorf("stdin is empty")
	}

	var args map[string]any
	if err := json.Unmarshal(data, &args); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return args, nil
}

// enhanceParseError adds flag hints to parse errors (e.g. missing required flags).
func enhanceParseError(err error, serverName string, tool *mcp.Tool) error {
	msg := err.Error()

	// Build flag summary for the hint.
	required := make(map[string]bool)
	for _, r := range tool.InputSchema.Required {
		required[r] = true
	}

	var flags []string
	propNames := sortedKeys(tool.InputSchema.Properties)
	for _, name := range propNames {
		prop := tool.InputSchema.Properties[name]
		entry := fmt.Sprintf("--%s (%s)", name, flagTypeLabel(prop.Type))
		if required[name] {
			entry += " *required*"
		}
		flags = append(flags, entry)
	}

	return fmt.Errorf("%s\n\nAvailable flags for %s:\n  %s\n\nRun: mcpx %s %s --help",
		msg, tool.Name, strings.Join(flags, "\n  "), serverName, tool.Name)
}

// parseToolFlags builds flags from a tool's JSON schema and parses rawArgs.
// Required flags are enforced.
func parseToolFlags(tool *mcp.Tool, rawArgs []string) (map[string]any, error) {
	return parseToolFlagsInternal(tool, rawArgs, false)
}

// parseToolFlagsPartial is like parseToolFlags but does not enforce required flags.
// Used when merging CLI flags on top of --stdin JSON.
func parseToolFlagsPartial(tool *mcp.Tool, rawArgs []string) (map[string]any, error) {
	return parseToolFlagsInternal(tool, rawArgs, true)
}

func parseToolFlagsInternal(tool *mcp.Tool, rawArgs []string, skipRequired bool) (map[string]any, error) {
	tmpCmd := &cobra.Command{
		Use:           tool.Name,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, args []string) error { return nil },
	}

	type flagValue struct {
		kind     string
		strVal   *string
		intVal   *int64
		floatVal *float64
		boolVal  *bool
	}

	flags := make(map[string]*flagValue)

	propNames := make([]string, 0, len(tool.InputSchema.Properties))
	for name := range tool.InputSchema.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	for _, name := range propNames {
		prop := tool.InputSchema.Properties[name]
		fv := &flagValue{kind: prop.Type}
		flags[name] = fv

		switch prop.Type {
		case "string":
			fv.strVal = new(string)
			tmpCmd.Flags().StringVar(fv.strVal, name, "", prop.Description)
		case "integer":
			fv.intVal = new(int64)
			tmpCmd.Flags().Int64Var(fv.intVal, name, 0, prop.Description)
		case "number":
			fv.floatVal = new(float64)
			tmpCmd.Flags().Float64Var(fv.floatVal, name, 0, prop.Description)
		case "boolean":
			fv.boolVal = new(bool)
			tmpCmd.Flags().BoolVar(fv.boolVal, name, false, prop.Description)
		case "array", "object":
			fv.strVal = new(string)
			tmpCmd.Flags().StringVar(fv.strVal, name, "", prop.Description+" (JSON)")
		default:
			fv.strVal = new(string)
			tmpCmd.Flags().StringVar(fv.strVal, name, "", prop.Description)
		}
	}

	if !skipRequired {
		for _, req := range tool.InputSchema.Required {
			if _, ok := flags[req]; ok {
				tmpCmd.MarkFlagRequired(req)
			}
		}
	}

	tmpCmd.SetArgs(rawArgs)
	if err := tmpCmd.Execute(); err != nil {
		return nil, err
	}

	result := make(map[string]any)
	for name, fv := range flags {
		if !tmpCmd.Flags().Changed(name) {
			continue
		}

		switch fv.kind {
		case "string":
			val, err := resolveStringValue(*fv.strVal, name)
			if err != nil {
				return nil, err
			}
			result[name] = val
		case "integer":
			result[name] = *fv.intVal
		case "number":
			result[name] = *fv.floatVal
		case "boolean":
			result[name] = *fv.boolVal
		case "array":
			val, err := resolveStringValue(*fv.strVal, name)
			if err != nil {
				return nil, err
			}
			var arr []any
			if err := json.Unmarshal([]byte(val), &arr); err != nil {
				// Fallback: comma-separated strings.
				parts := strings.Split(val, ",")
				arr = make([]any, len(parts))
				for i, p := range parts {
					arr[i] = strings.TrimSpace(p)
				}
			}
			result[name] = arr
		case "object":
			val, err := resolveStringValue(*fv.strVal, name)
			if err != nil {
				return nil, err
			}
			var obj map[string]any
			if err := json.Unmarshal([]byte(val), &obj); err != nil {
				return nil, fmt.Errorf("flag --%s: expected JSON object: %w", name, err)
			}
			result[name] = obj
		default:
			result[name] = *fv.strVal
		}
	}

	return result, nil
}

// connectServer resolves variables, connects to the MCP server, and runs lifecycle hooks.
// Routes to the appropriate transport based on config: http, sse, daemon, or stdio.
func connectServer(ctx context.Context, name string, sc *config.ServerConfig) (*mcp.Client, func(), error) {
	timeout := 30 * time.Second
	if sc.StartupTimeout != "" {
		if d, parseErr := time.ParseDuration(sc.StartupTimeout); parseErr == nil {
			timeout = d
		}
	}

	var client *mcp.Client
	var cleanup func()
	var err error

	switch sc.Transport {
	case "http":
		client, cleanup, err = connectHTTP(ctx, name, sc, timeout)
	case "sse":
		client, cleanup, err = connectSSE(ctx, name, sc, timeout)
	default:
		client, cleanup, err = connectStdio(ctx, name, sc, timeout)
	}
	if err != nil {
		return nil, nil, err
	}

	// Run lifecycle hooks (e.g. activate_project for Serena).
	if err := runLifecycleHooks(ctx, client, name, sc); err != nil {
		cleanup()
		return nil, nil, err
	}

	return client, cleanup, nil
}

// connectHTTP connects via the Streamable HTTP transport.
func connectHTTP(ctx context.Context, name string, sc *config.ServerConfig, timeout time.Duration) (*mcp.Client, func(), error) {
	headers, err := resolveHeaders(sc)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: %w", name, err)
	}

	url, err := resolveURL(sc)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: %w", name, err)
	}

	transport := mcp.NewHTTPTransport(url, headers)
	client := mcp.NewClient(transport)

	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Initialize(initCtx); err != nil {
		transport.Close()
		return nil, nil, fmt.Errorf("server %q: initialize: %w", name, err)
	}

	cleanup := func() { client.Close() }
	return client, cleanup, nil
}

// connectSSE connects via the legacy SSE transport.
func connectSSE(ctx context.Context, name string, sc *config.ServerConfig, timeout time.Duration) (*mcp.Client, func(), error) {
	headers, err := resolveHeaders(sc)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: %w", name, err)
	}

	url, err := resolveURL(sc)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: %w", name, err)
	}

	// Use the parent context for the SSE stream — not a short-lived timeout context,
	// since the SSE body reader stays open for the lifetime of the transport.
	transport, err := mcp.NewSSETransport(ctx, url, headers)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: sse connect: %w", name, err)
	}

	client := mcp.NewClient(transport)

	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Initialize(initCtx); err != nil {
		transport.Close()
		return nil, nil, fmt.Errorf("server %q: initialize: %w", name, err)
	}

	cleanup := func() { client.Close() }
	return client, cleanup, nil
}

// connectStdio connects via stdio (direct or daemon mode).
func connectStdio(ctx context.Context, name string, sc *config.ServerConfig, timeout time.Duration) (*mcp.Client, func(), error) {
	resolvedArgs, resolvedEnv, err := resolveServerConfig(sc)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: %w", name, err)
	}

	envSlice := make([]string, 0, len(resolvedEnv))
	for k, v := range resolvedEnv {
		envSlice = append(envSlice, k+"="+v)
	}

	// Daemon mode: connect via unix socket to a long-running process.
	if sc.Daemon {
		socketPath, err := daemon.EnsureRunning(ctx, name, sc.Command, resolvedArgs, envSlice, timeout)
		if err != nil {
			return nil, nil, fmt.Errorf("server %q: daemon: %w", name, err)
		}

		transport, err := daemon.NewSocketTransport(socketPath)
		if err != nil {
			return nil, nil, fmt.Errorf("server %q: connect daemon: %w", name, err)
		}

		client := mcp.NewClient(transport)

		// Initialize through the daemon — it returns its cached InitializeResult,
		// so the client gets server info and capabilities.
		initCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := client.Initialize(initCtx); err != nil {
			transport.Close()
			return nil, nil, fmt.Errorf("server %q: daemon initialize: %w", name, err)
		}

		cleanup := func() { transport.Close() } // closes socket, daemon stays alive
		return client, cleanup, nil
	}

	// Direct mode: spawn a fresh subprocess.
	transport, err := mcp.NewStdioTransport(sc.Command, resolvedArgs, envSlice)
	if err != nil {
		return nil, nil, fmt.Errorf("server %q: start: %w", name, err)
	}

	client := mcp.NewClient(transport)

	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Initialize(initCtx); err != nil {
		transport.Close()
		return nil, nil, fmt.Errorf("server %q: initialize: %w", name, err)
	}

	cleanup := func() { client.Close() }
	return client, cleanup, nil
}

// logAudit writes an audit log entry if audit logging is configured.
func logAudit(globalSec *config.SecurityConfig, serverName, toolName string, args map[string]any, result security.Result) {
	if globalSec == nil || globalSec.Global.Audit == nil || !globalSec.Global.Audit.Enabled {
		return
	}

	audit := globalSec.Global.Audit

	// Resolve variables in log path.
	logPath := audit.Log
	projectRoot, _ := findProjectRoot()
	secrets := secret.NewKeyringStore()
	res := resolver.New(projectRoot, secrets)
	if resolved, err := res.Resolve(logPath); err == nil {
		logPath = resolved
	}

	logger := security.NewAuditLogger(logPath, audit.Redact)
	_ = logger.Log(security.AuditEntry{
		Server:     serverName,
		Tool:       toolName,
		Args:       args,
		Action:     security.ActionString(result.Action),
		PolicyName: result.PolicyName,
		Message:    result.Message,
	})
}

// runLifecycleHooks executes on_connect hooks for a server if configured.
// If workspaces are defined and cwd is inside one, workspace hooks are used instead.
func runLifecycleHooks(ctx context.Context, client *mcp.Client, serverName string, sc *config.ServerConfig) error {
	projectRoot, _ := findProjectRoot()

	// Determine which hooks to run: workspace-specific or server-level.
	hooks := getActiveHooks(sc, projectRoot)
	if len(hooks) == 0 {
		return nil
	}

	// Resolve $(var) patterns in hook args.
	secrets := secret.NewKeyringStore()
	res := resolver.New(projectRoot, secrets)

	resolvedHooks := make([]config.LifecycleHook, len(hooks))
	for i, hook := range hooks {
		resolvedHooks[i] = config.LifecycleHook{
			Tool: hook.Tool,
			Args: make(map[string]any, len(hook.Args)),
		}
		for k, v := range hook.Args {
			if strVal, ok := v.(string); ok {
				resolved, err := res.Resolve(strVal)
				if err != nil {
					return fmt.Errorf("server %q: lifecycle hook %q: resolve arg %q: %w", serverName, hook.Tool, k, err)
				}
				resolvedHooks[i].Args[k] = resolved
			} else {
				resolvedHooks[i].Args[k] = v
			}
		}
	}

	return lifecycle.RunOnConnect(ctx, client, serverName, resolvedHooks)
}

// getActiveHooks returns the lifecycle hooks to execute based on cwd.
// If cwd is inside a workspace, workspace hooks are used. Otherwise, server-level hooks.
func getActiveHooks(sc *config.ServerConfig, projectRoot string) []config.LifecycleHook {
	// Check for workspace match.
	ws := config.ResolveWorkspace(sc, projectRoot)
	if ws != nil && ws.Lifecycle != nil && len(ws.Lifecycle.OnConnect) > 0 {
		return ws.Lifecycle.OnConnect
	}

	// Fall back to server-level hooks.
	if sc.Lifecycle != nil {
		return sc.Lifecycle.OnConnect
	}

	return nil
}

// resolveHeaders builds the HTTP headers map from config, including auth.
func resolveHeaders(sc *config.ServerConfig) (map[string]string, error) {
	projectRoot, _ := findProjectRoot()
	secrets := secret.NewKeyringStore()
	res := resolver.New(projectRoot, secrets)

	headers := make(map[string]string)
	for k, v := range sc.Headers {
		resolved, err := res.Resolve(v)
		if err != nil {
			return nil, fmt.Errorf("resolve header %q: %w", k, err)
		}
		headers[k] = resolved
	}

	// Apply auth config.
	if sc.Auth != nil && sc.Auth.Type == "bearer" && sc.Auth.Token != "" {
		token, err := res.Resolve(sc.Auth.Token)
		if err != nil {
			return nil, fmt.Errorf("resolve auth token: %w", err)
		}
		headers["Authorization"] = "Bearer " + token
	}

	return headers, nil
}

// resolveURL resolves dynamic variables in the server URL.
func resolveURL(sc *config.ServerConfig) (string, error) {
	projectRoot, _ := findProjectRoot()
	secrets := secret.NewKeyringStore()
	res := resolver.New(projectRoot, secrets)
	return res.Resolve(sc.URL)
}

// resolveServerConfig resolves $(var) patterns in a server config.
func resolveServerConfig(sc *config.ServerConfig) ([]string, map[string]string, error) {
	projectRoot, _ := findProjectRoot()
	secrets := secret.NewKeyringStore()
	res := resolver.New(projectRoot, secrets)

	resolvedArgs, err := res.ResolveSlice(sc.Args)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve args: %w", err)
	}

	resolvedEnv, err := res.ResolveMap(sc.Env)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve env: %w", err)
	}

	return resolvedArgs, resolvedEnv, nil
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys(m map[string]mcp.PropertySchema) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// findProjectRoot walks up from cwd looking for .mcpx/ directory.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if info, err := os.Stat(dir + "/.mcpx"); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := dir[:strings.LastIndex(dir, string(os.PathSeparator))]
		if parent == dir || parent == "" {
			return dir, nil
		}
		dir = parent
	}
}
