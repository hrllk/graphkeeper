package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	commitinspectoradapter "hrllk/graphkeeper/internal/adapter/out/commitinspector"
	gitpull "hrllk/graphkeeper/internal/adapter/out/gitpull"
	gitread "hrllk/graphkeeper/internal/adapter/out/gitread"
	tagprovenance "hrllk/graphkeeper/internal/adapter/out/tagprovenance"
	telemetryadapter "hrllk/graphkeeper/internal/adapter/out/telemetry"
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

	eventSink := telemetryadapter.New(filepath.Join(os.TempDir(), "graphkeeper-events.jsonl"))
	repo, err := git.OpenWithEventSink(".", eventSink)
	if err != nil {
		return 1, err
	}

	model, err := app.NewWithDependencies(app.Dependencies{Repo: repo, RepositoryRead: gitread.New(repo), Pull: gitpull.New(repo), InspectorReader: commitinspectoradapter.New(repo), TagProvenance: tagprovenance.New(repo.Root()), EventSink: eventSink})
	if err != nil {
		return 1, err
	}

	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}
