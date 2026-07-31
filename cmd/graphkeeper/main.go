package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"hrllk/graphkeeper/internal/app"
	"hrllk/graphkeeper/internal/git"
)

// version is overridden by release builds with -ldflags.
var version = "dev"

const usage = `Usage: graphkeeper [options]

Graph-first Git TUI for inspecting repository history and branch state.

Options:
  -h, --help       Show this help and exit
      --version    Show the version and exit

Run graphkeeper from inside a Git repository to start the TUI.
`

func main() {
	exitCode, err := execute(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func execute(args []string, stdout, stderr io.Writer) (int, error) {
	flags := flag.NewFlagSet("graphkeeper", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	showHelp := false
	showVersion := false
	flags.BoolVar(&showHelp, "help", false, "show help")
	flags.BoolVar(&showHelp, "h", false, "show help")
	flags.BoolVar(&showVersion, "version", false, "show version")

	if err := flags.Parse(args); err != nil {
		return 2, fmt.Errorf("%w\n\n%s", err, usage)
	}
	if flags.NArg() > 0 {
		return 2, fmt.Errorf("unexpected argument %q\n\n%s", flags.Arg(0), usage)
	}
	if showHelp {
		_, _ = io.WriteString(stdout, usage)
		return 0, nil
	}
	if showVersion {
		_, _ = fmt.Fprintf(stdout, "graphkeeper %s\n", version)
		return 0, nil
	}

	repo, err := git.Open(".")
	if err != nil {
		return 1, err
	}

	model, err := app.New(repo)
	if err != nil {
		return 1, err
	}

	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}
