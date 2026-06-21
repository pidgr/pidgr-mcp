// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package oauth

import (
	"fmt"
	"os/exec"
	"runtime"
)

// BrowserOpener opens the given URL in the user's default browser. It is an
// injection point so tests can drive the flow without launching a browser.
type BrowserOpener func(url string) error

// openBrowser launches the OS default browser at url.
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, *bsd
		cmd = "xdg-open"
		args = []string{url}
	}

	// G204: launching the OS browser handler with the authorize URL is the
	// intended behavior of an RFC 8252 native-app flow; the command name is a
	// fixed per-OS constant, only the URL we constructed varies.
	if err := exec.Command(cmd, args...).Start(); err != nil { //nolint:gosec // G204: fixed command, self-constructed URL
		return fmt.Errorf("open browser (%s): %w", cmd, err)
	}
	return nil
}
