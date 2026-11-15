package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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

// reloginDelay stands in for the 5 seconds the scripts wait before they
// log back in, acctDelay for the second they wait to send the account,
// and loginWatch for the minute they give a login before they zap it, so
// the checks take seconds. loginWatch has to outlast the slowest login
// here, about 2 seconds on Windows, or it would hide a missing limit on
// tries at the account. tickSecs is how often the scripts' tickers run
// here, so that one would go off during every login if it could send
// anything before the character is in.
const (
	reloginDelay = 500 * time.Millisecond
	acctDelay    = 300 * time.Millisecond
	loginWatch   = 5 * time.Second
	tickSecs     = "0.2"
)

// What TorilMUD sends at login, colour codes and all. A prompt has no
// newline after it.
const (
	acctPrompt = "\x1b[0m\x1b[37mAccount name: "
	noAccount  = "That account does not exist. Please enter an account " +
		"or \"new\" to create a new account\r\n" + acctPrompt
	pwPrompt = "\r\nEntering compressed mode.\r\nPassword: "
	wrongPw  = "\r\nInvalid password. Please try again.\r\n" + acctPrompt
	menuRule = "-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-" +
		"=-=-=-=-=-=-=-=-=-=-=-"
	charMenu = "\r\n\r\nCharacters:\r\n" + menuRule + "\r\n" +
		"  1) Testbot                   Level  1 Bard (Half-Elf)\r\n" +
		menuRule + "\r\n" +
		"(E)nter description    | (D)elete character | (C)hange password\r\n" +
		menuRule + "\r\n\r\nYour choice: "
)

// reconnect runs the script's own login and reconnect events against a
// stand-in for TorilMUD on 127.0.0.1. The MUD is down when tintin starts
// and refuses the login and the first relogin. It comes up for the next
// relogin, and then puts the script through what can go wrong at login:
// a garbled first line, prompts that come before the answer to the last,
// an account turned down three times, and a login that stalls. Each
// answer has to wait for its prompt and go out once, only the password
// prompt may get the password, and the script's ticker may send nothing
// until the character is in.
//
// Logging in straight from a reconnect event nests each failed retry
// inside the last until tt++ runs out of stack, so the script has to log
// back in through a #delay, and logging in again right after the drop
// fails the test.
func (s tinScript) reconnect(t *testing.T, tt, bin, work string) {
	// the MUD's port, which nothing listens on until the MUD comes up
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	chkErr(t, err)
	addr := ln.Addr().String()
	chkErr(t, ln.Close())
	tin, run := s.startLoopback(t, tt, bin, work, "reconnect", addr)
	defer tin.stop()

	// The MUD is down: the login when tintin starts fails, and so does
	// the relogin after it.
	if !tin.waitFor("TIMED OUT.", 2) {
		t.Fatalf("tintin didn't try twice while the MUD was down%s",
			tin.report())
	}

	// It comes up, and the next relogin gets in. The MUD asks for the
	// account twice before it comes, which has to get it once, and then
	// turns it down, as when the first line is garbled: the account has
	// to come again, not the password.
	ln, err = net.Listen("tcp", addr)
	chkErr(t, err)
	defer ln.Close()
	go func() { // stop waiting on a tt++ that has died
		<-tin.exited
		ln.Close()
	}()
	m := tin.connection(t, ln, "once the MUD was up")
	acct := m.ask(acctPrompt+noAccount, s.login[0],
		"two account prompts at once")
	m.quiet(2*acctDelay, "after answering two account prompts")
	m.ask(noAccount, acct, "the account prompt after the account was "+
		"turned down")
	m.ask(pwPrompt, s.login[1], "the password prompt")
	m.ask(charMenu, s.login[2], "the character menu")

	// It drops the character, who waits, then logs in again. The MUD asks
	// for the account again and for the password at once, as when it has
	// read a stray line and then the account: the password has to come,
	// and the account no more.
	dropped := time.Now()
	m.c.Close()
	m = tin.connection(t, ln, "again after the drop")
	if wait := time.Since(dropped); wait < reloginDelay/2 {
		t.Errorf("tintin logged in again %v after the drop, without "+
			"waiting out the relogin delay", wait.Round(time.Millisecond))
	}
	m.ask(acctPrompt, s.login[0], "the account prompt")
	m.ask(noAccount+pwPrompt, s.login[1],
		"an account prompt and the password prompt at once")
	m.quiet(2*acctDelay, "after the password")
	m.ask(charMenu, s.login[2], "the character menu")
	m.c.Close()

	// It logs in again, but the MUD turns the account down three times:
	// the script has to give up on that login.
	m = tin.connection(t, ln, "again after the second drop")
	acct = m.ask(acctPrompt, s.login[0], "the account prompt")
	m.ask(noAccount, acct, "the second account prompt")
	m.ask(noAccount, acct, "the third account prompt")
	m.send(noAccount, "the account turned down a third time")
	m.closes("after the account was turned down three times")
	m.c.Close()

	// It logs in again, and the MUD goes quiet after the account: the
	// script has to give up on that login too, after a while.
	m = tin.connection(t, ln, "again after giving up on the account")
	asked := time.Now()
	m.ask(acctPrompt, s.login[0], "the account prompt")
	m.closes("when the MUD went quiet after the account")
	if wait := time.Since(asked); wait < loginWatch/2 {
		t.Errorf("tintin gave up on the login %v after the account "+
			"prompt, without waiting for the password prompt",
			wait.Round(time.Millisecond))
	}
	m.c.Close()

	// It logs in again, and its ticker starts once the character is in.
	m = tin.connection(t, ln, "again after giving up on a quiet login")
	defer m.c.Close()
	m.login(s.login)
	m.waitLine(s.tick, "from the ticker once the character was in")

	// The driver quits when the MUD says so.
	m.send("TEST OVER\r\n", "the end of the test")
	select {
	case <-tin.exited:
	case <-time.After(20 * time.Second):
		t.Fatalf("tt++ didn't quit%s", tin.report())
	}
	if tin.err != nil {
		t.Fatalf("tt++: %v%s", tin.err, tin.report())
	}
	tin.checkScreen(t)
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

// password logs the script in to a stand-in MUD that turns down the
// password twice. The script may try it a second time, in case a stray
// line was taken for it, but then has to close the connection and stay
// out: trying a wrong password over and over can only lock the account.
func (s tinScript) password(t *testing.T, tt, bin, work string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	chkErr(t, err)
	defer ln.Close()
	tin, _ := s.startLoopback(t, tt, bin, work, "password",
		ln.Addr().String())
	defer tin.stop()
	go func() { // stop waiting on a tt++ that has died
		<-tin.exited
		ln.Close()
	}()

	m := tin.connection(t, ln, "when it started")
	defer m.c.Close()
	acct := m.ask(acctPrompt, s.login[0], "the account prompt")
	m.ask(pwPrompt, s.login[1], "the password prompt")
	m.ask(wrongPw, acct, "the account prompt after a wrong password")
	m.ask(pwPrompt, s.login[1], "the second password prompt")
	m.send(wrongPw, "the second wrong password")
	m.closes("after the password was turned down twice")

	// It stays out.
	chkErr(t, ln.(*net.TCPListener).SetDeadline(
		time.Now().Add(4*reloginDelay)))
	if c, err := ln.Accept(); err == nil {
		c.Close()
		t.Fatalf("tintin logged in again after the password was turned "+
			"down twice%s", tin.report())
	}
	select {
	case <-tin.exited:
		t.Fatalf("tt++ quit%s", tin.report())
	default:
	}
	if !strings.Contains(tin.screen(), "password twice") {
		t.Errorf("tintin didn't say why it stopped logging in%s",
			tin.report())
	}
	tin.checkScreen(t)
}

// startLoopback starts tt++ in a run folder of its own, named for what,
// on a copy of the script that logs in to the stand-in MUD at addr. It
// returns the folder too.
func (s tinScript) startLoopback(t *testing.T, tt, bin, work, what,
	addr string) (*tinRun, string) {
	run := filepath.Join(work,
		strings.TrimSuffix(s.name, ".tin")+"-"+what)
	chkErr(t, os.MkdirAll(filepath.Join(run, "logs"), 0750))
	copyLoopback(t, s.path, filepath.Join(run, s.name), addr)
	for _, dep := range s.deps {
		copyOffline(t, filepath.Join(filepath.Dir(s.path), dep),
			filepath.Join(run, dep))
	}
	writeTokens(t, run)
	chkErr(t, os.WriteFile(filepath.Join(run, "test.tin"),
		[]byte(s.reconnectDriver()), 0640))
	return startTintin(t, tt, bin, run), run
}

// reconnectDriver is the test.tin for reconnect and password: it loads the
// script, logs in at the start if the script leaves that to another file,
// and quits when the stand-in MUD says the test is over. It has no #end,
// so tt++ keeps running.
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

// relogin is a #delay that logs back in, up to its seconds, sendacct the
// #delay that sends the account, and loginwatch the #delay that gives up
// on a login that stalls.
var (
	relogin = regexp.MustCompile(
		`(?i)(#delay\s*\{[^{}]*\}\s*\{login\}\s*\{)[0-9.]+\}`)
	sendacct = regexp.MustCompile(
		`(?i)(#delay\s*\{sendacct\}\s*\{[^{}]*\}\s*\{)[0-9.]+\}`)
	loginwatch = regexp.MustCompile(
		`(?i)(#delay\s*\{loginwatch\}\s*\{[^{}]*\}\s*\{)[0-9.]+\}`)
)

// tickStart starts a #tick, and tickEnd ends one, up to its seconds.
var (
	tickStart = regexp.MustCompile(`(?i)^\s*#tick\b`)
	tickEnd   = regexp.MustCompile(`(\}\s*\{)[0-9.]+(\}\s*)$`)
)

// copyLoopback copies a tintin script so that it logs in to addr instead
// of TorilMUD, logs back in reloginDelay after a disconnect or a failed
// connect, sends the account acctDelay after its prompt, gives up on a
// login after loginWatch, and runs its tickers every tickSecs. Other
// #session lines, like a commented-out one for torilmud.org, are left
// out, and the test stops if anything is left that could connect anywhere
// else.
func copyLoopback(t *testing.T, from, to, addr string) {
	b, err := os.ReadFile(from)
	chkErr(t, err)
	host, port, err := net.SplitHostPort(addr)
	chkErr(t, err)
	loopback := host + " " + port
	secs := func(d time.Duration) string {
		return strconv.FormatFloat(d.Seconds(), 'f', -1, 64)
	}
	var keep []string
	sessions, delays, sends, watches, inTick := 0, 0, 0, 0, false
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
			line = relogin.ReplaceAllString(line,
				"${1}"+secs(reloginDelay)+"}")
			delays++
		}
		if sendacct.MatchString(line) {
			line = sendacct.ReplaceAllString(line,
				"${1}"+secs(acctDelay)+"}")
			sends++
		}
		if loginwatch.MatchString(line) {
			line = loginwatch.ReplaceAllString(line,
				"${1}"+secs(loginWatch)+"}")
			watches++
		}
		inTick = inTick || tickStart.MatchString(line)
		if inTick && tickEnd.MatchString(line) {
			line = tickEnd.ReplaceAllString(line, "${1}"+tickSecs+"${2}")
			inTick = false
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
	if sends == 0 || watches == 0 {
		t.Fatalf("%s doesn't send the account through a #delay {sendacct} "+
			"{...} {seconds} and give up on a login that stalls through "+
			"a #delay {loginwatch} {...} {seconds}", from)
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

// checkScreen fails the test if tintin couldn't read the scripts cleanly
// or closed a session twice, and logs its other errors and warnings.
func (r *tinRun) checkScreen(t *testing.T) {
	t.Helper()
	for _, line := range strings.Split(r.screen(), "\n") {
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

// mudConn is the stand-in MUD's end of a connection from tintin.
type mudConn struct {
	t  *testing.T
	r  *tinRun
	c  net.Conn
	in *bufio.Reader
}

// connection waits for tintin to connect to the stand-in MUD, which it
// has to do when.
func (r *tinRun) connection(t *testing.T, ln net.Listener,
	when string) *mudConn {
	t.Helper()
	c, err := accept(ln)
	if err != nil {
		t.Fatalf("tintin didn't log in %s: %v%s", when, err, r.report())
	}
	return &mudConn{t: t, r: r, c: c, in: bufio.NewReader(c)}
}

// send sends text, which is what.
func (m *mudConn) send(text, what string) {
	m.t.Helper()
	chkErr(m.t, m.c.SetDeadline(time.Now().Add(20*time.Second)))
	if _, err := io.WriteString(m.c, text); err != nil {
		m.t.Fatalf("sending %s: %v%s", what, err, m.r.report())
	}
}

// ask sends prompt, which is what, and returns the line tintin answers it
// with. That has to be want, or any line but an empty one if want is "".
// Only the password prompt may get the password.
func (m *mudConn) ask(prompt, want, what string) string {
	m.t.Helper()
	m.send(prompt, what)
	line, err := m.in.ReadString('\n')
	if err != nil {
		m.t.Fatalf("no answer to %s: %v%s", what, err, m.r.report())
	}
	switch line = strings.TrimRight(line, "\r\n"); {
	case line == fakeSecret && want != fakeSecret:
		m.t.Fatalf("tintin sent the password in answer to %s%s", what,
			m.r.report())
	case line == "":
		m.t.Fatalf("tintin answered %s with an empty line%s", what,
			m.r.report())
	case want != "" && line != want:
		m.t.Fatalf("tintin answered %s with %q, not %q%s", what, line,
			want, m.r.report())
	}
	return line
}

// login answers a login's prompts in turn: want holds the account ("" for
// any), the password and the character.
func (m *mudConn) login(want []string) {
	m.t.Helper()
	m.ask(acctPrompt, want[0], "the account prompt")
	m.ask(pwPrompt, want[1], "the password prompt")
	m.ask(charMenu, want[2], "the character menu")
}

// waitLine reads what tintin sends, for up to 5 seconds, until the line
// want comes, which it has to send when.
func (m *mudConn) waitLine(want, when string) {
	m.t.Helper()
	chkErr(m.t, m.c.SetDeadline(time.Now().Add(5*time.Second)))
	for {
		line, err := m.in.ReadString('\n')
		if err != nil {
			m.t.Fatalf("tintin didn't send %q %s: %v%s", want, when, err,
				m.r.report())
		}
		if strings.TrimRight(line, "\r\n") == want {
			return
		}
	}
}

// quiet fails the test if tintin sends anything for d, which is when.
func (m *mudConn) quiet(d time.Duration, when string) {
	m.t.Helper()
	chkErr(m.t, m.c.SetDeadline(time.Now().Add(d)))
	line, err := m.in.ReadString('\n')
	if line != "" || !errors.Is(err, os.ErrDeadlineExceeded) {
		m.t.Fatalf("tintin sent %q %s (%v)%s", line, when, err,
			m.r.report())
	}
}

// closes waits up to 20 seconds for tintin to close the connection, which
// it has to do when, and fails if it sends anything first.
func (m *mudConn) closes(when string) {
	m.t.Helper()
	chkErr(m.t, m.c.SetDeadline(time.Now().Add(20*time.Second)))
	line, err := m.in.ReadString('\n')
	switch {
	case line != "":
		m.t.Fatalf("tintin sent %q %s%s", line, when, m.r.report())
	case errors.Is(err, os.ErrDeadlineExceeded):
		m.t.Fatalf("tintin didn't close the connection %s%s", when,
			m.r.report())
	}
}
