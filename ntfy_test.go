package main

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Test_Ntfy checks the reboot alert gobot posts to the topic in ntfy.url, the
// test alert gobot -testalert sends there, and that an error from ntfy, or a
// missing or empty ntfy.url, comes back as one.
func Test_Ntfy(t *testing.T) {
	type post struct{ to, title, msg string }
	posts := make(chan post, 4)
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			msg, err := io.ReadAll(r.Body)
			chkErr(t, err)
			posts <- post{r.Method + " " + r.URL.Path,
				r.Header.Get("Title"), string(msg)}
			if r.URL.Path == "/busy" {
				// what ntfy.sh answers past its limits
				http.Error(w, `{"code":42901,"http":429,`+
					`"error":"limit reached: too many requests"}`,
					http.StatusTooManyRequests)
			}
		}))
	defer srv.Close()

	src, err := os.Getwd()
	chkErr(t, err)
	chkErr(t, os.Chdir(t.TempDir()))
	defer os.Chdir(src)
	chkErr(t, os.WriteFile(ntfyFile, []byte(srv.URL+"/katumi\n"), 0600))

	// posted checks what was posted since the last check, if anything
	posted := func(check string, want *post) {
		t.Helper()
		select {
		case got := <-posts:
			if want == nil {
				t.Errorf("%s check failed: posted %+v", check, got)
			} else if got != *want {
				t.Errorf("%s check failed: Expected: %+v, Actual: %+v",
					check, *want, got)
			}
		default:
			if want != nil {
				t.Errorf("%s check failed: nothing posted", check)
			}
		}
	}

	SendBootAlert()
	posted("SendBootAlert()", &post{"POST /katumi", "TorilMUD reboot/crash",
		"Katumi detected a new TorilMUD boot."})

	chkErr(t, SendTestAlert())
	posted("SendTestAlert()", &post{"POST /katumi", "TorilMUD alert test",
		"A test from gobot -testalert. Reboot alerts will arrive here too."})

	err = ntfy(srv.URL+"/busy", "TorilMUD reboot/crash", "test")
	if err == nil || !strings.Contains(err.Error(), "429 Too Many Requests") ||
		!strings.Contains(err.Error(), "limit reached") {
		t.Errorf("ntfy() check failed: a 429 gave %v", err)
	}
	posted("ntfy()", &post{"POST /busy", "TorilMUD reboot/crash", "test"})

	// without a topic, the test alert says why rather than stopping gobot,
	// so -testalert can print it; the topic has to be on the first line
	chkErr(t, os.WriteFile(ntfyFile, []byte("\n"+srv.URL+"/katumi\n"), 0600))
	err = SendTestAlert()
	if err == nil || !strings.Contains(err.Error(), "no ntfy topic URL") {
		t.Errorf("SendTestAlert() check failed: a blank first line gave %v",
			err)
	}
	posted("SendTestAlert() with a blank first line", nil)
	chkErr(t, os.Remove(ntfyFile))
	if err = SendTestAlert(); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("SendTestAlert() check failed: no %s gave %v", ntfyFile, err)
	}
	posted("SendTestAlert() without "+ntfyFile, nil)
}
