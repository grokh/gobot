package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Test_Ntfy checks the reboot alert gobot posts to the topic in ntfy.url, and
// that an error from ntfy comes back as one.
func Test_Ntfy(t *testing.T) {
	type post struct{ to, title, msg string }
	posts := make(chan post, 2)
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

	SendBootAlert()
	want := post{"POST /katumi", "TorilMUD reboot/crash",
		"Katumi detected a new TorilMUD boot."}
	select {
	case got := <-posts:
		if got != want {
			t.Errorf("SendBootAlert() check failed: Expected: %+v, "+
				"Actual: %+v", want, got)
		}
	default:
		t.Errorf("SendBootAlert() check failed: nothing posted")
	}

	err = ntfy(srv.URL+"/busy", "TorilMUD reboot/crash", "test")
	if err == nil || !strings.Contains(err.Error(), "429 Too Many Requests") ||
		!strings.Contains(err.Error(), "limit reached") {
		t.Errorf("ntfy() check failed: a 429 gave %v", err)
	}
}
