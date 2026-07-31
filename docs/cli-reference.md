# CLI reference

`graphkeeper` is a Git TUI. Run it from inside a Git repository to inspect
history, refs, remotes, tags, and stash state.

## Invocation

```text
graphkeeper [options]
```

The command accepts options before starting the TUI. Help and version output do
not require the current directory to be a Git repository.

## Options

| Option | Effect | Exit code |
| --- | --- | ---: |
| `-h` | Print the usage text and supported options. | `0` |
| `--help` | Print the usage text and supported options. | `0` |
| `--version` | Print `graphkeeper <version>`. | `0` |

If more than one supported option is supplied, help takes precedence over
version output. For example, `graphkeeper --help --version` prints help and
exits successfully.

## Errors

| Input or condition | Behavior | Exit code |
| --- | --- | ---: |
| Unknown option, such as `--unknown` | Print the parser error and usage text to stderr. | `2` |
| Positional argument, such as `graphkeeper repo` | Print an unexpected-argument error and usage text to stderr. | `2` |
| Git/app startup error after option parsing | Print the error to stderr. | `1` |

Option parsing happens before the repository is opened. This allows these
commands to work from any directory:

```bash
graphkeeper --help
graphkeeper --version
```

Starting the TUI still requires a usable Git repository. The application opens
the current directory, then loads repository state when the TUI initializes.

## Version resolution

The source default is `dev`:

```text
graphkeeper dev
```

Release builds override this value with Go's linker flags. The repository's
[`scripts/build`](../scripts/build) script reads [`VERSION`](../VERSION) and
passes the value as `-X main.version=<version>`.

```bash
./scripts/build /tmp/graphkeeper
/tmp/graphkeeper --version
```

When using `go build` directly, pass the linker flag yourself if the binary
must report a release version:

```bash
go build -ldflags "-X main.version=alpha.5" -o graphkeeper ./cmd/graphkeeper
```

## Implementation surface

The CLI is implemented in [`cmd/graphkeeper/main.go`](../cmd/graphkeeper/main.go).
`execute` receives argument and output streams, which keeps option handling
testable without launching Bubble Tea or requiring a repository.

Related tests are in
[`cmd/graphkeeper/main_test.go`](../cmd/graphkeeper/main_test.go).

## Related

- [How to build and run graphkeeper](howto-build-and-run.md)
- [README quick start](../README.md#quick-start)
