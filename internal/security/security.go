package security

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/codestz/mcpx/v2/internal/config"
)

// Action represents the result of a policy evaluation.
type Action int

const (
	// ActionAllow permits the tool call.
	ActionAllow Action = iota
	// ActionDeny blocks the tool call.
	ActionDeny
	// ActionWarn permits the call but emits a warning.
	ActionWarn
)

// Result holds the outcome of evaluating security policies.
type Result struct {
	Action     Action
	PolicyName string
	Message    string
	Details    string // extra context (e.g. which arg matched)
}

// Evaluator evaluates security policies for a given server.
type Evaluator struct {
	serverName string
	mode       string
	allowed    []string
	blocked    []string
	global     []config.Policy
	server     []config.Policy
}

// NewEvaluator creates an Evaluator from global and per-server security config.
func NewEvaluator(serverName string, global *config.SecurityConfig, server *config.ServerSecurity) *Evaluator {
	e := &Evaluator{serverName: serverName}

	if global != nil {
		e.global = global.Global.Policies
	}
	if server != nil {
		e.mode = server.Mode
		e.allowed = server.AllowedTools
		e.blocked = server.BlockedTools
		e.server = server.Policies
	}

	return e
}

// Evaluate checks whether a tool call is permitted.
func (e *Evaluator) Evaluate(toolName string, args map[string]any) Result {
	// 1. Check mode-level restrictions.
	if r := e.checkMode(toolName); r.Action == ActionDeny {
		return r
	}

	// 2. Check allowed/blocked tool lists.
	if r := e.checkToolLists(toolName); r.Action == ActionDeny {
		return r
	}

	// 3. Evaluate global policies.
	for _, p := range e.global {
		if r := evaluatePolicy(e.serverName, p, toolName, args); r.Action != ActionAllow {
			return r
		}
	}

	// 4. Evaluate server-level policies.
	for _, p := range e.server {
		if r := evaluatePolicy(e.serverName, p, toolName, args); r.Action != ActionAllow {
			return r
		}
	}

	return Result{Action: ActionAllow}
}

// checkMode applies mode-based restrictions.
// "read-only" denies tools that are typically write operations.
func (e *Evaluator) checkMode(toolName string) Result {
	if e.mode != "read-only" {
		return Result{Action: ActionAllow}
	}

	// Common write tool patterns.
	writePatterns := []string{
		"replace_*", "insert_*", "delete_*", "remove_*",
		"create_*", "update_*", "rename_*", "write_*",
		"execute", "drop_*", "alter_*",
	}

	for _, pattern := range writePatterns {
		if matched, _ := filepath.Match(pattern, toolName); matched {
			return Result{
				Action:     ActionDeny,
				PolicyName: "(read-only mode)",
				Message:    fmt.Sprintf("server %q: read-only mode denied tool %q", e.serverName, toolName),
				Details:    "Hint: set security.mode to \"editing\" or \"custom\" to allow write tools",
			}
		}
	}

	return Result{Action: ActionAllow}
}

// checkToolLists checks allowed/blocked tool whitelists and blacklists.
func (e *Evaluator) checkToolLists(toolName string) Result {
	// If allowed list is set, tool must match at least one pattern.
	if len(e.allowed) > 0 {
		matched := false
		for _, pattern := range e.allowed {
			if m, _ := filepath.Match(pattern, toolName); m {
				matched = true
				break
			}
		}
		if !matched {
			return Result{
				Action:     ActionDeny,
				PolicyName: "(allowed_tools)",
				Message:    fmt.Sprintf("server %q: tool %q not in allowed_tools list", e.serverName, toolName),
			}
		}
	}

	// If blocked list is set, tool must not match any pattern.
	for _, pattern := range e.blocked {
		if m, _ := filepath.Match(pattern, toolName); m {
			return Result{
				Action:     ActionDeny,
				PolicyName: "(blocked_tools)",
				Message:    fmt.Sprintf("server %q: tool %q is in blocked_tools list", e.serverName, toolName),
			}
		}
	}

	return Result{Action: ActionAllow}
}

// evaluatePolicy checks a single policy against a tool call.
func evaluatePolicy(serverName string, p config.Policy, toolName string, args map[string]any) Result {
	// Check tool name match.
	if len(p.Match.Tools) > 0 {
		matched := false
		for _, pattern := range p.Match.Tools {
			if m, _ := filepath.Match(pattern, toolName); m {
				matched = true
				break
			}
		}
		if !matched {
			return Result{Action: ActionAllow} // policy doesn't apply
		}
	}

	// Check arg rules.
	if len(p.Match.Args) > 0 {
		if r := checkArgRules(serverName, p, args); r.Action != ActionAllow {
			return r
		}
	}

	// Check content match.
	if p.Match.Content != nil {
		if r := checkContentMatch(serverName, p, args); r.Action != ActionAllow {
			return r
		}
	}

	// If we have arg or content rules but none triggered, allow.
	if len(p.Match.Args) > 0 || p.Match.Content != nil {
		return Result{Action: ActionAllow}
	}

	// Policy with only tool match and no arg/content rules: apply the action.
	return policyResult(serverName, p, toolName, "")
}

// checkArgRules evaluates argument-level rules in a policy.
func checkArgRules(serverName string, p config.Policy, args map[string]any) Result {
	for argPattern, rule := range p.Match.Args {
		// Find matching arg names (supports glob patterns like "*path*").
		for argName, argVal := range args {
			matched, _ := filepath.Match(argPattern, argName)
			if !matched {
				continue
			}

			strVal := fmt.Sprintf("%v", argVal)

			// Check deny_pattern.
			if rule.DenyPattern != "" {
				if re, err := regexp.Compile(rule.DenyPattern); err == nil && re.MatchString(strVal) {
					return policyResult(serverName, p, "", fmt.Sprintf("%s = %q", argName, strVal))
				}
			}

			// Check allow_prefix — value must start with one of the prefixes.
			if len(rule.AllowPrefix) > 0 {
				allowed := false
				for _, prefix := range rule.AllowPrefix {
					if strings.HasPrefix(strVal, prefix) {
						allowed = true
						break
					}
				}
				if !allowed {
					return policyResult(serverName, p, "", fmt.Sprintf("%s = %q", argName, strVal))
				}
			}

			// Check deny_prefix — value must not start with any of the prefixes.
			for _, prefix := range rule.DenyPrefix {
				if strings.HasPrefix(strVal, prefix) {
					return policyResult(serverName, p, "", fmt.Sprintf("%s = %q", argName, strVal))
				}
			}
		}
	}

	return Result{Action: ActionAllow}
}

// checkContentMatch inspects arg values with regex patterns.
func checkContentMatch(serverName string, p config.Policy, args map[string]any) Result {
	cm := p.Match.Content

	// Extract target arg value (e.g. "args.sql" → args["sql"]).
	argName := cm.Target
	if trimmed, ok := strings.CutPrefix(argName, "args."); ok {
		argName = trimmed
	}

	val, ok := args[argName]
	if !ok {
		return Result{Action: ActionAllow} // arg not present, rule doesn't apply
	}
	strVal := fmt.Sprintf("%v", val)

	// Check "when" condition — only apply rules if the value matches "when".
	if cm.When != "" {
		whenRe, err := regexp.Compile(cm.When)
		if err != nil || !whenRe.MatchString(strVal) {
			return Result{Action: ActionAllow}
		}
	}

	// Check deny_pattern.
	if cm.DenyPattern != "" {
		if re, err := regexp.Compile(cm.DenyPattern); err == nil && re.MatchString(strVal) {
			return policyResult(serverName, p, "", fmt.Sprintf("content in %s matched deny pattern", cm.Target))
		}
	}

	// Check require_pattern — if not present, trigger action.
	if cm.RequirePattern != "" {
		if re, err := regexp.Compile(cm.RequirePattern); err == nil && !re.MatchString(strVal) {
			return policyResult(serverName, p, "", fmt.Sprintf("content in %s missing required pattern", cm.Target))
		}
	}

	return Result{Action: ActionAllow}
}

// policyResult converts a policy action string to a Result.
func policyResult(_ string, p config.Policy, _ string, detail string) Result {
	action := ActionDeny
	switch p.Action {
	case "allow":
		action = ActionAllow
	case "warn":
		action = ActionWarn
	}

	msg := p.Message
	if msg == "" {
		msg = fmt.Sprintf("policy %q triggered", p.Name)
	}

	return Result{
		Action:     action,
		PolicyName: p.Name,
		Message:    msg,
		Details:    detail,
	}
}
