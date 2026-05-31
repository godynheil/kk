# Testing Guide

This guide explains how testing is implemented in the KK project and how to write effective tests. The project uses Go's built-in `testing` package with a focus on table-driven tests, integration tests, and reusable test utilities.

---

## Testing Framework

KK uses Go's standard `testing` package from the standard library. No external testing frameworks are required. For comprehensive Go testing documentation, see the [official Go testing guide](https://go.dev/doc/tutorial/add-a-test).

---

## Test Organization

The codebase contains 24 test files organized by package:

### internal/app/

Application layer tests covering the full range of KK commands:

- `accounts_test.go` - Google Drive account management
- `add_test.go` - File staging and tracking
- `app_test.go` - Core application logic
- `checkout_test.go` - Branch switching with tracked assets
- `clone_test.go` - Repository cloning workflows
- `init_test.go` - Repository initialization
- `objects_test.go` - Object reference management
- `project_test.go` - Project management and re-importing
- `pull_test.go` - Pull operations and merge detection
- `push_test.go` - Push operations and remote validation
- `remote_test.go` - Remote management and migration
- `setup_test.go` - Setup and OAuth flows
- `sync_test.go` - Multi-remote object synchronization
- `testhelpers_test.go` - Common test utilities (see [Test Utilities](#test-utilities-and-helpers))

### internal/core/

Core business logic tests:

- `code_test.go` - Code language detection
- `history_test.go` - Git history, bundle name, and branch validation

### internal/remote/

Remote storage integration tests:

- `concurrent_test.go` - Concurrent file processing and worker pools
- `drive_test.go` - Google Drive integration and retry logic
- `path_security_test.go` - Path sanitization and traversal prevention
- `rclone_test.go` - Rclone integration and content verification

### Other Packages

- `internal/ignore/matcher_test.go` - File ignore pattern matching
- `internal/registry/projects_test.go` - Project registry management
- `internal/git/client_test.go` - Git client wrapper functionality
- `internal/storage/local_test.go` - Local storage engine operations

---

## Test Patterns

### Table-Driven Tests

Table-driven tests are the most common pattern for testing multiple input combinations. This approach keeps tests concise and makes it easy to add new test cases.

```go
func TestIsValidBundleName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid full", "full.bundle", true},
		{"valid inc", "inc-000001.bundle", true},
		{"valid inc high sequence", "inc-999999.bundle", true},
		{"invalid prefix", "in-000001.bundle", false},
		{"invalid suffix", "inc-000001.bundl", false},
		{"invalid length", "inc-00001.bundle", false},
		{"invalid chars", "inc-000a01.bundle", false},
		{"empty", "", false},
		{"path traversal attempt", "../full.bundle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidBundleName(tt.input); got != tt.want {
				t.Errorf("IsValidBundleName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
```

> [!TIP]
> Use descriptive names for both the test function and subtest names. This makes test output more readable when debugging failures.

### Integration Tests

Integration tests set up complete environments with real filesystems, Git repositories, and application instances. They test end-to-end workflows.

```go
func TestPullSucceedsOnFastForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	// Set up repositories
	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))
	dirA := filepath.Join(tmp, "local-a")
	appA := initKKTestRepo(t, dirA)

	// Test logic here...
}
```

Key patterns for integration tests:

- Use `t.TempDir()` for isolated test directories that are automatically cleaned up
- Use `t.Setenv()` to configure environment variables for the test
- Skip with `if testing.Short() { t.Skip(...) }` for tests that take longer
- Create realistic test scenarios with multiple repositories

### Unit Tests

Unit tests verify individual functions with pure inputs and outputs, without external dependencies like filesystems or network calls.

```go
func TestDefaultPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"empty returns default", "", "/default/path"},
		{"absolute path unchanged", "/custom/path", "/custom/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultPath(tt.path); got != tt.want {
				t.Errorf("DefaultPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

---

## Test Utilities and Helpers

### Core Helpers

The file [`internal/app/testhelpers_test.go`](../internal/app/testhelpers_test.go) contains common helper functions used across integration tests:

| Function | Purpose |
| :--- | :--- |
| `mustGit(t, client, args...)` | Execute git commands, fail on error |
| `initKKTestRepo(t, dir)` | Initialize a test KK repository with git config |
| `initBareRepo(t, dir)` | Create a bare git repository for remotes |
| `writeAndCommit(t, gc, root, name, content, msg)` | Write a file and commit it |

### Helper Function Pattern

When writing helper functions:

1. Always call `t.Helper()` to mark the function as a test helper
2. Use `t.Fatal()` or `t.Fatalf()` for failures (not `t.Error()`)
3. Return values that tests can use for assertions

```go
func mustGit(t *testing.T, client git.Client, args ...string) {
	t.Helper()
	_, stderr, err := client.Combined(args...)
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, stderr)
	}
}
```

---

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run a specific test
go test -run TestPull

# Run tests in a specific package
go test ./internal/core/...

# Run with race detector
go test -race ./...
```

### Short Mode

Skip integration tests in short mode:

```bash
go test -short ./...
```

Integration tests should check for short mode and skip:

```go
if testing.Short() {
    t.Skip("skipping integration test in short mode")
}
```

### Makefile

The project Makefile provides a convenient target:

```bash
make test
```

---

## Test Naming Conventions

Follow these naming patterns for consistency:

- Test functions: `Test<FunctionName>` or `Test<Feature><Scenario>`
  - Examples: `TestIsValidBundleName`, `TestPullSucceedsOnFastForward`
- Subtests: Descriptive names indicating what is being tested
  - Examples: `"valid full"`, `"invalid prefix"`, `"path traversal attempt"`

Descriptive names make test output more readable when debugging failures.

---

## Best Practices

### Use t.Helper() in Helper Functions

Marking helpers with `t.Helper()` ensures that failure line numbers point to the test code, not the helper implementation.

### Use t.TempDir() for Isolation

`t.TempDir()` creates a temporary directory that is automatically removed after the test completes. This ensures tests don't interfere with each other.

### Test Both Success and Error Cases

For each function, test valid inputs, invalid inputs, edge cases, and error conditions.

```go
{"valid input", "good-value", true},
{"empty input", "", false},
{"invalid format", "bad!!value", false},
```

### Use Table-Driven Tests for Multiple Cases

When testing multiple input combinations, use table-driven tests rather than writing separate test functions.

### Skip Slow Tests in Short Mode

Integration tests that involve filesystem operations, Git commands, or network calls should skip in short mode to allow quick test runs during development.

### Write Descriptive Test Names

Test names should clearly indicate what is being tested and what the expected outcome is.

---

## Code Examples

### Complete Table-Driven Test Example

```go
func TestSafeRemoteRel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"simple relative path", "folder/file", false},
		{"absolute path rejected", "/etc/passwd", true},
		{"path traversal blocked", "../other", true},
		{"backslash rejected", "folder\\file", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}
```

### Integration Test with Helpers

```go
func TestCloneSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	// Create a bare repository to clone from
	bareDir := filepath.Join(tmp, "bare.git")
	initBareRepo(t, bareDir)

	// Clone the repository
	cloneDir := filepath.Join(tmp, "cloned")
	app := New(cloneDir)
	if err := app.Clone(bareDir); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// Verify the clone worked
	if _, err := os.Stat(filepath.Join(cloneDir, ".kk", "git")); err != nil {
		t.Errorf(".kk/git not created: %v", err)
	}
}
```

### Using Test Helpers

```go
func TestPushWithModifiedFiles(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	// Set up two repositories
	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))
	localDir := filepath.Join(tmp, "local")
	app := initKKTestRepo(t, localDir)
	gc := git.New(localDir)

	// Create and commit a test file
	writeAndCommit(t, gc, localDir, "test.txt", "content", "initial commit")

	// Test push logic...
}
```
