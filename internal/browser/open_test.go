package browser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// stubOpen puts a fake `open` at the front of PATH so the test drives the
// helper's exit status without talking to the real desktop.
func stubOpen(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "open")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestOpenReportsHelperFailure(t *testing.T) {
	stubOpen(t, "#!/bin/sh\necho 'procNotFound: no eligible process' >&2\nexit 1\n")

	err := Open("http://localhost:9999/")
	if err == nil {
		t.Fatal("expected an error when the helper exits non-zero")
	}
	// The helper's own stderr is what tells the user why, so it has to survive.
	if !strings.Contains(err.Error(), "procNotFound") {
		t.Errorf("error = %q, want it to carry the helper's stderr", err)
	}
}

func TestOpenSucceedsWhenHelperSucceeds(t *testing.T) {
	stubOpen(t, "#!/bin/sh\nexit 0\n")

	if err := Open("http://localhost:9999/"); err != nil {
		t.Errorf("Open() = %v, want nil", err)
	}
}
