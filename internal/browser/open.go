package browser

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Open opens the given URL in the default browser.
//
// It waits for the helper to exit rather than firing and forgetting. The helper
// hands the URL off to the desktop and returns immediately, so there is nothing
// to block on, and its exit status is the only signal that the handoff failed.
// Under a sandbox it routinely does: the browser is activated but the event
// carrying the URL is refused, which looks to the user like their browser came
// to the front and did nothing.
func Open(url string) error {
	cmd := exec.Command("open", url)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
		return err
	}
	return nil
}
