package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/codestz/mcpx/v2/internal/secret"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// secretCmd creates the "secret" command group for managing secrets in the OS keychain.
func secretCmd(opts *globalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets in the OS keychain",
	}

	store := secret.NewKeyringStore()

	cmd.AddCommand(secretSetCmd(store, opts))
	cmd.AddCommand(secretRemoveCmd(store, opts))
	cmd.AddCommand(secretListCmd(store, opts))

	return cmd
}

func secretSetCmd(store secret.Store, opts *globalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "set <name> <value>",
		Short: "Store a secret in the OS keychain",
		Long:  "Store a secret that can be referenced as $(secret.<name>) in server configs.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, value := args[0], args[1]

			if err := validateSecretName(name); err != nil {
				return err
			}

			if err := store.Set(name, value); err != nil {
				return fmt.Errorf("set secret: %w", err)
			}

			if opts.outputMode() != outputQuiet {
				out := newOutput(opts.outputMode())
				if opts.jsonOutput {
					return out.printJSON(map[string]any{
						"name":   name,
						"action": "set",
					})
				}
				green := color.New(color.FgGreen)
				green.Fprint(os.Stdout, "ok")
				fmt.Fprintf(os.Stdout, "  secret %q stored. Use $(secret.%s) in configs.\n", name, name)
			}

			return nil
		},
	}
}

func secretRemoveCmd(store secret.Store, opts *globalOpts) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a secret from the OS keychain",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if err := store.Delete(name); err != nil {
				return fmt.Errorf("remove secret: %w", err)
			}

			if opts.outputMode() != outputQuiet {
				out := newOutput(opts.outputMode())
				if opts.jsonOutput {
					return out.printJSON(map[string]any{
						"name":   name,
						"action": "removed",
					})
				}
				fmt.Fprintf(os.Stdout, "Removed secret %q\n", name)
			}

			return nil
		},
	}
}

func secretListCmd(store secret.Store, opts *globalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored secret names",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := store.List()
			if err != nil {
				return fmt.Errorf("list secrets: %w", err)
			}

			out := newOutput(opts.outputMode())

			if opts.jsonOutput {
				return out.printJSON(keys)
			}

			if len(keys) == 0 {
				fmt.Fprintln(out.stdout, "No secrets stored.")
				fmt.Fprintln(out.stdout, "Use: mcpx secret set <name> <value>")
				return nil
			}

			dim := color.New(color.FgHiBlack)
			for _, k := range keys {
				fmt.Fprintf(out.stdout, "  %s", k)
				dim.Fprintf(out.stdout, "  $(secret.%s)", k)
				fmt.Fprintln(out.stdout)
			}

			return nil
		},
	}
}

// validateSecretName ensures secret names are safe for use in $(secret.<name>) patterns.
func validateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-') {
			return fmt.Errorf("secret name %q contains invalid character %q. Use letters, digits, underscore, dash, or dot.", name, string(c))
		}
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "-") {
		return fmt.Errorf("secret name %q cannot start with %q", name, name[:1])
	}
	return nil
}
