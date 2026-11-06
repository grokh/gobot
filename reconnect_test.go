package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// reloginDelay stands in for the 30 seconds the scripts wait before they
// log back in, so the reconnect check takes seconds.
const reloginDelay = 500 * time.Millisecond

// reconnect runs the script's own login and reconnect events against a
// stand-in for TorilMUD on 127.0.0.1, through an outage. The MUD is down
// when tintin starts and refuses the login and the first relogin. It comes
// up for the next relogin, then drops the character, who has to wait out
// the relogin delay and log in again. Each login has to send the account,
// password and character, and only once the MUD asks for them.
//
// Logging in straight from a reconnect event nests each failed retry
// inside the last until tt++ runs out of stack, so the script has to log
// back in through a #delay, and logging in again right after the drop
// fails the test.
func (s tinScript) reconnect(t *testing.T, tt, bin, work string) {
	run := filepath.Join(work,
		strings.TrimSuffix(s.name, ".tin")+"-reconnect")
	chkErr(t, os.MkdirAll(filepath.Join(run, "logs"), 0750))
	// the MUD's port, which nothing listens on until the MUD comes up
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	chkErr(t, err)
	addr := ln.Addr().String()
	chkErr(t, ln.Close())

	copyLoopback(t, s.path, filepath.Join(run, s.name), addr)
	for _, dep := range s.deps {
		copyOffline(t, filepath.Join(filepath.Dir(s.path), dep),
			filepath.Join(run, dep))
	}
	writeTokens(t, run)
	chkErr(t, os.WriteFile(filepath.Join(run, "test.tin"),
		[]byte(s.reconnectDriver()), 0640))

	tin := startTintin(t, tt, bin, run)
	defer tin.stop()

	// The MUD is down: the login when tintin starts fails, and so does
	// the relogin after it.
	if !tin.waitFor("TIMED OUT.", 2) {
		t.Fatalf("tintin didn't try twice while the MUD was down%s",
			tin.report())
	}

	// It comes up, and the next relogin gets in.
	ln, err = net.Listen("tcp", addr)
	chkErr(t, err)
	defer ln.Close()
	go func() { // stop waiting on a tt++ that has died
		<-tin.exited
		ln.Close()
	}()
	c, err := accept(ln)
	if err != nil {
		t.Fatalf("tintin didn't log in once the MUD was up: %v%s", err,
			tin.report())
	}
	tin.login(t, c, s.login)

	// It drops the character, who waits, then logs in again.
	dropped := time.Now()
	c.Close()
	if c, err = accept(ln); err != nil {
		t.Fatalf("tintin didn't log in again after the drop: %v%s", err,
			tin.report())
	}
	defer c.Close()
	if wait := time.Since(dropped); wait < reloginDelay/2 {
		t.Errorf("tintin logged in again %v after the drop, without "+
			"waiting out the relogin delay", wait.Round(time.Millisecond))
	}
	tin.login(t, c, s.login)

	// The driver quits when the MUD says so.
	if _, err := io.WriteString(c, "TEST OVER\r\n"); err != nil {
		t.Fatalf("ending the test: %v%s", err, tin.report())
	}
	select {
	case <-tin.exited:
	case <-time.After(20 * time.Second):
		t.Fatalf("tt++ didn't quit%s", tin.report())
	}
	if tin.err != nil {
		t.Fatalf("tt++: %v%s", tin.err, tin.report())
	}
	for _, line := range strings.Split(tin.screen(), "\n") {
		switch line = strings.TrimSpace(line); {
		case strings.Contains(line, "#READ"):
			t.Errorf("tintin didn't read the scripts cleanly: %s", line)
		case strings.Contains(line, "ALREADY CLOSED"):
			t.Errorf("tintin closed a session twice: %s", line)
		case strings.HasPrefix(line, "#ERROR") ||
			strings.HasPrefix(line, "#WARNING"):
			t.Log(line)
		}
	}
	// each login opens the day's log of the session
	logs, err := filepath.Glob(filepath.Join(run, "logs", "*.log"))
	chkErr(t, err)
	logged := false
	for _, f := range logs {
		logged = logged || strings.Contains(readIfThere(t, f),
			"Account name:")
	}
	if !logged {
		t.Errorf("the logins weren't logged in logs/: %q", logs)
	}
}

// reconnectDriver is the test.tin for reconnect: it loads the script, logs
// in at the start if the script leaves that to another file, and quits
// when the stand-in MUD says the test is over. It has no #end, so tt++
// keeps running.
func (s tinScript) reconnectDriver() string {
	lines := []string{
		"#var {dir} {.}",
		"#act {^TEST OVER$} {#end}",
		"#read " + s.name,
		"#var {cpw} {" + fakeSecret + "}",
	}
	if s.start != "" {
		lines = append(lines, "#event {PROGRAM START} {"+s.start+"}")
	}
	return strings.Join(lines, "\n") + "\n"
}

// mudAddress is TorilMUD's host and port in a #session line.
var mudAddress = regexp.MustCompile(`(?i)\btorilmud\.com\s+9999\b`)

// relogin is a #delay that logs back in, up to its seconds.
var relogin = regexp.MustCompile(
	`(?i)(#delay\s*\{[^{}]*\}\s*\{login\}\s*\{)[0-9.]+\}`)

// copyLoopback copies a tintin script so that it logs in to addr instead
// of TorilMUD and logs back in reloginDelay after a disconnect or a failed
// connect. Other #session lines, like a commented-out one for
// torilmud.org, are left out, and the test stops if anything is left that
// could connect anywhere else.
func copyLoopback(t *testing.T, from, to, addr string) {
	b, err := os.ReadFile(from)
	chkErr(t, err)
	host, port, err := net.SplitHostPort(addr)
	chkErr(t, err)
	loopback := host + " " + port
	secs := strconv.FormatFloat(reloginDelay.Seconds(), 'f', -1, 64)
	var keep []string
	sessions, delays := 0, 0
	for _, line := range strings.Split(
		strings.ReplaceAll(string(b), "\r", ""), "\n") {
		if strings.Contains(strings.ToLower(line), "#ses") {
			if !mudAddress.MatchString(line) {
				continue
			}
			line = mudAddress.ReplaceAllLiteralString(line, loopback)
			sessions++
		}
		if relogin.MatchString(line) {
			line = relogin.ReplaceAllString(line, "${1}"+secs+"}")
			delays++
		}
		keep = append(keep, line)
	}
	if sessions == 0 {
		t.Fatalf("%s has no #session line for torilmud.com 9999", from)
	}
	if delays == 0 {
		t.Fatalf("%s doesn't log back in through a #delay {name} {login} "+
			"{seconds}: logging in straight from a reconnect event nests "+
			"each failed retry inside the last until tt++ runs out of stack",
			from)
	}
	txt := strings.Join(keep, "\n")
	toLoopback := regexp.MustCompile(
		`(?i)#ses\w*\s+\S+\s+` + regexp.QuoteMeta(loopback) + `\b`)
	if found := connects.FindString(
		toLoopback.ReplaceAllString(txt, "")); found != "" {
		t.Fatalf("%s has %s, which could connect elsewhere: not running it",
			from, found)
	}
	chkErr(t, os.WriteFile(to, []byte(txt), 0640))
}

// tinRun is tt++ running in the background.
type tinRun struct {
	cancel context.CancelFunc
	exited chan struct{} // closed once tt++ has exited and err is set
	err    error

	mu  sync.Mutex
	out bytes.Buffer
}

// Write takes what tt++ prints.
func (r *tinRun) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.out.Write(p)
}

// startTintin starts tt++ in dir on its test.tin.
func startTintin(t *testing.T, tt, bin, dir string) *tinRun {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	r := &tinRun{cancel: cancel, exited: make(chan struct{})}
	cmd := tintinCmd(ctx, tt, bin, dir)
	cmd.Stdout = r
	cmd.Stderr = r
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("tt++: %v", err)
	}
	go func() {
		r.err = cmd.Wait()
		close(r.exited)
	}()
	return r
}

// stop kills tt++ if it's still running, and waits for it to go.
func (r *tinRun) stop() {
	r.cancel()
	<-r.exited
}

// screen is what tt++ has printed so far, without colour codes.
func (r *tinRun) screen() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ansiCodes.ReplaceAllString(
		strings.ReplaceAll(r.out.String(), "\r", ""), "")
}

// report is how tt++ is doing, for the end of a failure message.
func (r *tinRun) report() string {
	state := "still running"
	select {
	case <-r.exited:
		state = "exited"
		if r.err != nil {
			state += ": " + r.err.Error()
		}
	default:
	}
	return "\ntt++ " + state + "\nscreen:\n" + r.screen()
}

// waitFor waits up to 20 seconds for tt++ to print want n times, and says
// whether it did.
func (r *tinRun) waitFor(want string, n int) bool {
	timeout := time.After(20 * time.Second)
	for strings.Count(r.screen(), want) < n {
		select {
		case <-r.exited:
			return strings.Count(r.screen(), want) >= n
		case <-timeout:
			return false
		case <-time.After(50 * time.Millisecond):
		}
	}
	return true
}

// accept waits up to 20 seconds for tintin to connect to the MUD.
func accept(ln net.Listener) (net.Conn, error) {
	err := ln.(*net.TCPListener).SetDeadline(time.Now().Add(20 * time.Second))
	if err != nil {
		return nil, err
	}
	return ln.Accept()
}

// login asks for an account the way TorilMUD does, and checks the lines
// that come back against want, where "" takes any line.
func (r *tinRun) login(t *testing.T, c net.Conn, want []string) {
	t.Helper()
	chkErr(t, c.SetDeadline(time.Now().Add(20*time.Second)))
	// TorilMUD's prompt, colour codes and all, with no newline after it
	if _, err := io.WriteString(c, "\x1b[0m\x1b[37mAccount name: "); err != nil {
		t.Fatalf("asking for an account: %v%s", err, r.report())
	}
	in := bufio.NewReader(c)
	for i, w := range want {
		line, err := in.ReadString('\n')
		if err != nil {
			t.Fatalf("no login line %d after the account prompt: %v%s",
				i+1, err, r.report())
		}
		switch line = strings.TrimRight(line, "\r\n"); {
		case line == "":
			t.Fatalf("login line %d is empty%s", i+1, r.report())
		case w != "" && line != w:
			t.Fatalf("login line %d is %q, not %q%s", i+1, line, w,
				r.report())
		}
	}
}
