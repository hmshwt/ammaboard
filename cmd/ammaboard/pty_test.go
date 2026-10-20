//go:build !windows

package main

// pty_test.go — the Go-native pty smoke (D44). Drives the real binary through
// a real pseudoterminal: rendering, the cursor highlight, run to halt, the
// shell, scroll keys on every pane, help, quit, and a clean exit code. This
// retires the Python scratch harness (tui_smoke.py in /tmp); unlike it, the
// expectations poll with a deadline instead of sleeping on faith, and the
// escape stripping reuses testrender.StripANSI.
//
// Assertions are cumulative-buffer contains — the point is that the real
// terminal path works end to end and never wedges, not that any single frame
// is exact (golden files own exactness, when they land with internal/tui).

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"ammaboard/internal/tuikit/testrender"
	"github.com/creack/pty"
)

// Input pacing — the only place where timing matters. The receiver (bubbletea
// v2 via ultraviolet's TerminalReader) holds a lone esc byte for
// DefaultEscTimeout (50ms, hardcoded there, not an option) waiting for a
// continuation byte: an esc followed by any keystroke inside that window
// parses as alt+<key>. No "flush" exists that a sender could send — the
// ambiguity lives in the receiver's parser — so pacing past the window is the
// only lever. This is the ncurses ESCDELAY convention; the constant is 2× the
// receiver's timeout for scheduler-jitter margin.
const escPace = 100 * time.Millisecond

// Output synchronization: expect polls the stream at this interval. It — not
// input pacing — is the real synchronization for every state assertion.
const (
	pollInterval  = 25 * time.Millisecond
	expectTimeout = 10 * time.Second
)

func TestTUIGoesThroughARealPty(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty on windows")
	}
	if testing.Short() {
		t.Skip("the pty smoke is the long path — run without -short")
	}

	bin, root := buildBinary(t)
	s := startTUI(t, bin, filepath.Join(root, "examples", "counter.amb"))

	// First paint: all three panes, the hints line, the machine not started.
	s.expect("add_number r1, r0, 0", "not started",
		"memory · f follows the pc", "tick log · f follows newest",
		"j/k scroll", "f follow")

	// The cursor highlight: a white-on-dark-green background escape must be
	// emitted (truecolor 48;2;0;100;0 for #006400, or a downgraded 48;5;N).
	s.expectRaw(`48;2;0;100;0|48;5;\d+`, "no dark-green cursor background escape")

	// Run to halt, then reset through the shell.
	s.send(" ")
	s.expect("clean halt")
	s.send(":")       // focus the shell; byte order keeps this ahead of "reset\r"
	s.send("reset\r") // run it
	s.expect("machine reset")
	s.esc() // blur the shell: pane keys from here on

	// Scroll the focused pane around: detach, jump, re-follow — no crashes.
	// The keys ride in one write: on a pty, byte order is processing order
	// and no plain key begins an escape sequence, so they cannot merge. The
	// expect() call is the synchronization — no pacing between keystrokes.
	s.send("jkGgf")
	s.expect("add_number")

	// Tab through all three panes (machine pane scrolls memory, numbers
	// scrolls the log) — focus cycling works on the real terminal.
	s.send("\tj\tj\tjk")
	s.expect("r0 00000000")

	// Help opens, esc closes it, toggling is harmless.
	s.send("?")
	s.expect("jump to top / bottom")
	s.esc() // esc closes the help overlay
	s.expect("memory · f follows the pc")

	// Quit — and the process must exit cleanly, code 0.
	s.send("??") // toggle help: open, close
	s.send("q")
	select {
	case err := <-s.exited:
		if err != nil {
			t.Fatalf("tui did not exit cleanly: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("tui did not exit within 5s of q")
	}
}

// buildBinary compiles the real binary (not go run: the pty child should be
// the artifact, with no wrapper between it and the terminal).
func buildBinary(t *testing.T) (bin, root string) {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	root = filepath.Dir(strings.TrimSpace(string(out)))
	bin = filepath.Join(t.TempDir(), "ammaboard")
	build := exec.Command("go", "build", "-o", bin, "./cmd/ammaboard")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin, root
}

// tuiSession owns one pty-hosted TUI process and a cumulative read buffer.
type tuiSession struct {
	t   *testing.T
	tty *os.File
	cmd *exec.Cmd

	mu   sync.Mutex
	buf  []byte
	raw  []byte
	done chan struct{} // closed when the read loop ends

	exited chan error // the single cmd.Wait() result
}

func startTUI(t *testing.T, bin, prog string) *tuiSession {
	t.Helper()
	cmd := exec.Command(bin, "tui", prog)
	// Pin the color vocabulary so the highlight assertion is deterministic:
	// truecolor if the stack honors it, 256-color at worst — never monochrome
	// just because the test host has no TERM.
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 30, Cols: 145})
	if err != nil {
		t.Fatalf("pty start: %v", err)
	}

	s := &tuiSession{
		t:      t,
		tty:    tty,
		cmd:    cmd,
		done:   make(chan struct{}),
		exited: make(chan error, 1),
	}
	go func() { s.exited <- cmd.Wait() }()
	go func() {
		defer close(s.done)
		chunk := make([]byte, 65536)
		for {
			n, err := tty.Read(chunk)
			if n > 0 {
				s.mu.Lock()
				s.raw = append(s.raw, chunk[:n]...)
				s.buf = append(s.buf, chunk[:n]...)
				s.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		tty.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-s.exited:
		case <-time.After(2 * time.Second):
		}
	})
	return s
}

// send writes literal bytes into the pty — the child reads them as keystrokes.
func (s *tuiSession) send(keys string) {
	s.t.Helper()
	if _, err := s.tty.Write([]byte(keys)); err != nil {
		s.t.Fatalf("write to pty: %v", err)
	}
}

// esc writes esc and paces it (see escPace): the next keystroke must arrive
// after the receiver's escape timeout, or it merges into alt+<key>.
func (s *tuiSession) esc() {
	s.send("\x1b")
	time.Sleep(escPace)
}

// plain is the escape-stripped output so far.
func (s *tuiSession) plain() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return testrender.StripANSI(string(s.buf))
}

// expect polls until every needle appears in the stripped output, or fails
// with the tail of what was actually rendered.
func (s *tuiSession) expect(needles ...string) {
	s.t.Helper()
	deadline := time.Now().Add(expectTimeout)
	for time.Now().Before(deadline) {
		plain := s.plain()
		missing := false
		for _, n := range needles {
			if !strings.Contains(plain, n) {
				missing = true
				break
			}
		}
		if !missing {
			return
		}
		time.Sleep(pollInterval)
	}
	plain := s.plain()
	tail := plain
	if len(tail) > 1200 {
		tail = tail[len(tail)-1200:]
	}
	s.t.Fatalf("not all expectations appeared within %s\nwant: %q\n---- tail ----\n%s",
		expectTimeout, needles, tail)
}

// expectRaw polls the *unstripped* stream for a regex — for assertions about
// the escape vocabulary itself (colors), not the visible text.
func (s *tuiSession) expectRaw(pattern, failMsg string) {
	s.t.Helper()
	re := regexp.MustCompile(pattern)
	deadline := time.Now().Add(expectTimeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		raw := string(s.raw)
		s.mu.Unlock()
		if re.MatchString(raw) {
			return
		}
		time.Sleep(pollInterval)
	}
	s.t.Fatal(failMsg)
}
