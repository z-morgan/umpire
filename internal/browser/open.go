package browser

import "os/exec"

// Open opens the given URL in the default browser.
func Open(url string) error {
	return exec.Command("open", url).Start()
}
