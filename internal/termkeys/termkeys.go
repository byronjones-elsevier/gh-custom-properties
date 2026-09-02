// Package termkeys optionally enables a terminal's Kitty keyboard protocol
// disambiguation flag, so Alt+letter keypresses arrive as a distinct key
// event on terminals that support it (kitty, WezTerm, Ghostty, and newer
// iTerm2 builds).
//
// This is purely additive: gh-custom-properties binds Alt+<letter>
// alongside every bare-letter command regardless of whether this succeeds,
// so terminals that don't support the protocol — including macOS's default
// Terminal.app, which sends neither this protocol nor a plain Alt/Meta
// escape sequence unless the user manually enables "Use Option as Meta
// Key" — are unaffected; their users simply never send the Alt+ sequence,
// and the bare letter keeps working everywhere.
package termkeys

import (
	"os"
	"time"

	"golang.org/x/term"
)

const (
	enableSeq  = "\x1b[=1;1b" // CSI = 1 ; 1 b: push and enable flag 1 (disambiguate escape codes)
	disableSeq = "\x1b[<1b"   // CSI < 1 b: pop the pushed flag state, restoring prior behavior
)

// EnableIfAvailable probes for Kitty keyboard protocol support and, if
// found, asks the terminal to enable disambiguated escape-code reporting.
// It reports whether it was enabled; callers should call Disable before
// exiting if so.
//
// It must run before anything else reads stdin — in particular, before
// tea.Program.Run() starts Bubble Tea's own input loop — since the probe
// briefly puts the terminal into raw mode and reads the response itself;
// doing this concurrently with another reader would race for the same
// bytes.
func EnableIfAvailable() (bool, error) {
	available, err := isKittyKeyboardAvailable()
	if err != nil || !available {
		return false, err
	}
	if _, err := os.Stdout.Write([]byte(enableSeq)); err != nil {
		return false, err
	}
	return true, nil
}

// Disable restores the terminal's prior keyboard-protocol flag state. Only
// meaningful, and safe, to call if EnableIfAvailable returned true.
func Disable() {
	_, _ = os.Stdout.Write([]byte(disableSeq))
}

// isKittyKeyboardAvailable queries the terminal for Kitty keyboard protocol
// support via CSI ? u, giving up after a short timeout on terminals that
// don't recognize the query and simply never reply.
func isKittyKeyboardAvailable() (bool, error) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return false, nil
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false, err
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	if _, err := os.Stdout.Write([]byte("\x1b[?u")); err != nil {
		return false, err
	}

	type readResult struct {
		buf []byte
		err error
	}
	resultCh := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 32)
		n, err := os.Stdin.Read(buf)
		resultCh <- readResult{buf: buf[:n], err: err}
	}()

	select {
	case r := <-resultCh:
		if r.err != nil {
			return false, nil
		}
		return len(r.buf) >= 3 && r.buf[0] == '\x1b' && r.buf[1] == '[' && r.buf[len(r.buf)-1] == 'u', nil
	case <-time.After(100 * time.Millisecond):
		return false, nil
	}
}
