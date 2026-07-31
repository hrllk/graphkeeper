# How to build and run graphkeeper

Build a versioned `graphkeeper` binary, verify its CLI output, and start the
TUI inside a Git repository.

## Prerequisites

- Go version declared in [`go.mod`](../go.mod)
- A Git repository for the TUI run
- A writable working directory for the build cache and output binary

## Steps

1. Set the application version in [`VERSION`](../VERSION).

   ```bash
   printf 'alpha.5\n' > VERSION
   ```

   Use the release version you want users to see. The build script removes
   surrounding whitespace before injecting it into the binary.

2. Build the binary with the repository script.

   ```bash
   ./scripts/build
   ```

   The default output is `./graphkeeper`. To choose another path:

   ```bash
   ./scripts/build /tmp/graphkeeper
   ```

3. Check the CLI before entering a repository.

   ```bash
   ./graphkeeper --help
   ./graphkeeper --version
   ```

   Help and version output should succeed even when the current directory is
   not a Git repository. A release build reports the value from `VERSION`, for
   example `graphkeeper alpha.5`.

4. Change to a Git repository and start the TUI.

   ```bash
   cd /path/to/repository
   /path/to/graphkeeper
   ```

   The TUI loads the graph and repository context from the current directory.

## Verification

Run the automated checks from the project root:

```bash
scripts/check
```

The CLI-specific tests cover help, version output, and rejection of unknown
options. You can also verify the non-success path manually:

```bash
set +e
./graphkeeper --unknown >/tmp/graphkeeper.stdout 2>/tmp/graphkeeper.stderr
status=$?
set -e
test "$status" -eq 2
grep -q 'Usage: graphkeeper \[options\]' /tmp/graphkeeper.stderr
```

## Troubleshooting

### `graphkeeper --version` prints `graphkeeper dev`

The binary was built without a release linker value. Rebuild with
`./scripts/build`, or pass `-ldflags "-X main.version=<version>"` to `go build`.

### Starting the TUI fails outside a Git repository

Run the binary from a Git working tree. Use `--help` or `--version` when you
only need CLI information and do not have a repository available.

### An option is rejected

Only `-h`, `--help`, and `--version` are supported. The parser returns exit code
`2` for unknown options and positional arguments, then prints the usage text.

## Related

- [CLI reference](cli-reference.md)
- [README quick start](../README.md#quick-start)
