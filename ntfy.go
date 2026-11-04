package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ntfyFile, in the bot's folder, has the ntfy topic that reboot alerts go to,
// as a URL on its first line, ex. https://ntfy.sh/<topic>. It stays out of
// the repo: anyone who knows the topic can read the alerts and send their own.
const ntfyFile = "ntfy.url"

// SendBootAlert tells everyone subscribed to the ntfy topic that TorilMUD has
// rebooted or crashed.
func SendBootAlert() {
	content, err := os.ReadFile(ntfyFile)
	ChkErr(err)
	topic := strings.TrimSpace(strings.SplitN(string(content), "\n", 2)[0])
	if topic == "" {
		fatal("Fatal Error: no ntfy topic URL on the first line of", ntfyFile)
	}
	msg := "Katumi detected a new TorilMUD boot."
	log.Printf("Alert: %s\n", msg)
	ChkErr(ntfy(topic, "TorilMUD reboot/crash", msg))
}

// ntfy posts a notification to an ntfy topic URL.
func ntfy(topic string, title string, msg string) error {
	req, err := http.NewRequest("POST", topic, strings.NewReader(msg))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	// give up, rather than leave a gobot -time waiting on ntfy for good
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		// ntfy says why in a line of JSON
		why, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ntfy: %s: %s",
			resp.Status, strings.TrimSpace(string(why)))
	}
	return nil
}
