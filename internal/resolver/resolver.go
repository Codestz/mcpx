package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/codestz/mcpx/internal/secret"
)

// varPattern matches $(namespace.key) with strict validation.
var varPattern = regexp.MustCompile(`\$\(([a-z]+)\.([a-zA-Z0-9_.]+)\)`)

// NamespaceResolver resolves keys within a namespace.
type NamespaceResolver interface {
	Resolve(key string) (string, error)
}

// Resolver holds all namespace resolvers and resolves variable strings.
type Resolver struct {
	namespaces map[string]NamespaceResolver
}

// New creates a Resolver with all built-in namespace resolvers.
func New(projectRoot string, secrets secret.Store) *Resolver {
	r := &Resolver{namespaces: make(map[string]NamespaceResolver)}
	r.namespaces["mcpx"] = &mcpxNS{projectRoot: projectRoot}
	r.namespaces["git"] = &gitNS{projectRoot: projectRoot}
	r.namespaces["env"] = &envNS{}
	r.namespaces["secret"] = &secretNS{store: secrets}
	r.namespaces["sys"] = &sysNS{}
	return r
}

// NewWithResolvers creates a Resolver with the given namespace resolvers.
// Useful for testing with mock resolvers.
func NewWithResolvers(resolvers map[string]NamespaceResolver) *Resolver {
	return &Resolver{namespaces: resolvers}
}

// Resolve replaces all $(namespace.key) patterns in a string.
func (r *Resolver) Resolve(s string) (string, error) {
	var resolveErr error
	result := varPattern.ReplaceAllStringFunc(s, func(match string) string {
		if resolveErr != nil {
			return match
		}
		sub := varPattern.FindStringSubmatch(match)
		if len(sub) != 3 {
			resolveErr = fmt.Errorf("resolve: invalid variable %q", match)
			return match
		}
		ns, key := sub[1], sub[2]
		resolver, ok := r.namespaces[ns]
		if !ok {
			resolveErr = fmt.Errorf("resolve: unknown namespace %q in %q", ns, match)
			return match
		}
		val, err := resolver.Resolve(key)
		if err != nil {
			resolveErr = fmt.Errorf("resolve %q: %w", match, err)
			return match
		}
		return val
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}

// ResolveSlice resolves all strings in a slice.
func (r *Resolver) ResolveSlice(ss []string) ([]string, error) {
	out := make([]string, len(ss))
	for i, s := range ss {
		resolved, err := r.Resolve(s)
		if err != nil {
			return nil, fmt.Errorf("resolve slice [%d]: %w", i, err)
		}
		out[i] = resolved
	}
	return out, nil
}

// ResolveMap resolves all values in a map.
func (r *Resolver) ResolveMap(m map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(m))
	for k, v := range m {
		resolved, err := r.Resolve(v)
		if err != nil {
			return nil, fmt.Errorf("resolve map key %q: %w", k, err)
		}
		out[k] = resolved
	}
	return out, nil
}

// --- mcpx namespace ---

type mcpxNS struct {
	projectRoot string
}

func (n *mcpxNS) Resolve(key string) (string, error) {
	switch key {
	case "project_root":
		return filepath.Clean(n.projectRoot), nil
	case "cwd":
		return os.Getwd()
	case "home":
		return os.UserHomeDir()
	default:
		return "", fmt.Errorf("mcpx: unknown key %q", key)
	}
}

// --- git namespace ---

type gitNS struct {
	projectRoot string
}

func (n *gitNS) Resolve(key string) (string, error) {
	var args []string
	switch key {
	case "root":
		args = []string{"rev-parse", "--show-toplevel"}
	case "branch":
		args = []string{"rev-parse", "--abbrev-ref", "HEAD"}
	case "remote":
		args = []string{"remote", "get-url", "origin"}
	case "commit":
		args = []string{"rev-parse", "HEAD"}
	default:
		return "", fmt.Errorf("git: unknown key %q", key)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = n.projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// --- env namespace ---

type envNS struct{}

func (n *envNS) Resolve(key string) (string, error) {
	return os.Getenv(key), nil
}

// --- secret namespace ---

type secretNS struct {
	store secret.Store
}

func (n *secretNS) Resolve(key string) (string, error) {
	val, err := n.store.Get(key)
	if err != nil {
		return "", fmt.Errorf("secret %q: %w", key, err)
	}
	return val, nil
}

// --- sys namespace ---

type sysNS struct{}

func (n *sysNS) Resolve(key string) (string, error) {
	switch key {
	case "os":
		return runtime.GOOS, nil
	case "arch":
		return runtime.GOARCH, nil
	default:
		return "", fmt.Errorf("sys: unknown key %q", key)
	}
}
