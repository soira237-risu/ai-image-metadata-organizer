# imv CLI Hardening Implementation Plan

> **For agentic workers:** Execute inline in this session with the `superpowers:test-driven-development` workflow; multi-agent dispatch is not in scope for this run.

**Goal:** Make command-specific help exit successfully and reject CLI input that is currently ignored or silently coerced.

**Architecture:** Keep command dispatch in `cmd/imv/main.go`, but centralize command synopses and flag-help handling so every subcommand behaves consistently. Validate syntax at the CLI boundary before opening SQLite or touching the filesystem; leave the shared `appcore` APIs unchanged.

**Tech Stack:** Go 1.22 module, standard-library `flag`, table-driven Go tests.

## Global Constraints

- Preserve the existing command names, defaults, JSON schemas, and dry-run move behavior.
- Add no runtime dependency.
- Write each behavior test first and observe the expected failure before production changes.
- Do not modify the user's existing untracked cache or generated files.

---

### Task 1: Command-specific help

**Files:**
- Modify: `cmd/imv/main_test.go`
- Modify: `cmd/imv/main.go`

**Interfaces:**
- Consumes: existing `runWithIO(args, stdout, stderr) error` test boundary.
- Produces: `imv <command> --help` and `imv help <command>` output beginning with `Usage: imv <command>` on stdout, with a nil error.

- [x] **Step 1: Write failing help tests**

```go
func TestSubcommandHelpFlagPrintsCommandUsageToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runWithIO([]string{"scan", "--help"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	for _, want := range []string{"Usage: imv scan", "-workers"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote stderr: %s", stderr.String())
	}
}

func TestHelpCommandPrintsSubcommandUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runWithIO([]string{"help", "search"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage: imv search") {
		t.Fatalf("unexpected help output:\n%s", stdout.String())
	}
}
```

- [x] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/imv -run 'Test(SubcommandHelpFlag|HelpCommand)' -count=1`

Expected: `scan --help` returns `flag: help requested`, and `help search` prints only general help.

- [x] **Step 3: Implement shared help handling**

Add usage constants for all seven subcommands, route `help <command>` back through the relevant command with `--help`, suppress the standard library's stderr help callback, and add:

```go
func parseCommandFlags(fs *flag.FlagSet, args []string, stdout io.Writer, usage string) (bool, error) {
	err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandUsage(stdout, usage, fs)
		return true, nil
	}
	return false, err
}

func printCommandUsage(stdout io.Writer, usage string, fs *flag.FlagSet) {
	fmt.Fprintf(stdout, "Usage: %s\n\nOptions:\n", usage)
	fs.SetOutput(stdout)
	fs.PrintDefaults()
}
```

Each command returns immediately when the helper reports that help was printed. Change `runExport` to receive stdout so its help follows the same contract.

- [x] **Step 4: Run the focused tests and verify GREEN**

Run: `go test ./cmd/imv -run 'Test(SubcommandHelpFlag|HelpCommand)' -count=1`

Expected: both tests pass with no stderr output.

### Task 2: Strict CLI argument validation

**Files:**
- Modify: `cmd/imv/main_test.go`
- Modify: `cmd/imv/main.go`

**Interfaces:**
- Consumes: parsed `flag.FlagSet` values before an `appcore.Service` is created.
- Produces: usage errors for unexpected positional arguments and explicit errors for non-positive `--workers`/`--limit` values.

- [x] **Step 1: Write failing validation tests**

```go
func TestCommandsRejectUnexpectedPositionalArguments(t *testing.T) {
	tests := [][]string{
		{"search", "unexpected"},
		{"tags", "unexpected"},
		{"stats", "unexpected"},
		{"export", "unexpected", "--out", "out.json"},
		{"move", "unexpected", "--tag", "blue hair", "--to", "dest"},
	}
	for _, args := range tests {
		err := runWithIO(args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "usage: imv "+args[0]) {
			t.Fatalf("%v: expected usage error, got %v", args, err)
		}
	}
}

func TestCommandsRejectNonPositiveNumericOptions(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"scan", "images", "--workers", "0"}, "--workers must be greater than zero"},
		{[]string{"search", "--limit", "0"}, "--limit must be greater than zero"},
		{[]string{"tags", "--limit", "-1"}, "--limit must be greater than zero"},
	}
	for _, tt := range tests {
		err := runWithIO(tt.args, &bytes.Buffer{}, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%v: expected %q, got %v", tt.args, tt.want, err)
		}
	}
}
```

- [x] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cmd/imv -run 'TestCommandsReject' -count=1`

Expected: commands with extra arguments proceed to database or file work, and zero values are silently converted to defaults.

- [x] **Step 3: Implement boundary validation**

After help handling and before calling `appcore`, require `NArg() == 0` for `search`, `tags`, `stats`, `export`, and `move`. Reuse each command's usage constant in the returned error. Require `workers > 0` for `scan` and `limit > 0` for `search`/`tags`:

```go
if fs.NArg() != 0 {
	return fmt.Errorf("usage: %s", searchUsage)
}
if *limit <= 0 {
	return fmt.Errorf("--limit must be greater than zero")
}
```

- [x] **Step 4: Run focused and full verification**

Run: `go test ./cmd/imv -run 'TestCommandsReject' -count=1`

Expected: all validation tests pass.

Run: `go test ./...`

Expected: all Go packages pass.

Run: `go vet ./...`

Expected: exit code 0 with no diagnostics.

Run: `go build -o .\\bin\\imv.exe .\\cmd\\imv`

Expected: exit code 0.

Run: `.\\bin\\imv.exe help scan`

Expected: command-specific scan usage and flags on stdout with exit code 0.
