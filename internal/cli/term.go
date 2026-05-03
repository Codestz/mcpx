package cli

import (
	"os"

	"golang.org/x/term"
)

func isStderrTerminal() bool {
	return term.IsTerminal(int(os.Stderr.Fd()))
}
