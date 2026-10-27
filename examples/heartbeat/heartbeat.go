// Package heartbeat is a miniature of the EntMon daemon's shape: one struct,
// snake_case methods, JSON state on disk, a string status machine, and a
// sequential poll loop. It records whether a URL responded, and it will not
// re-record a success until the caller clears state.
package heartbeat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// pulse_record is one attempt against the URL. It is appended after every
// poll (success or failure) and written to disk immediately so a crash
// cannot lose the fact that we already succeeded.
type pulse_record struct {
	Checked_at  string `json:"checked_at"` // RFC3339 UTC
	Status      string `json:"status"`     // "success" or "failed"
	Status_code int    `json:"status_code"`
}

// persisted_state is the schema of the JSON file on disk. Keeping it in a
// named type (rather than an anonymous struct at the call site) means the
// on-disk format has one definition a Python reader can search for.
type persisted_state struct {
	Target string         `json:"target"`
	Pulses []pulse_record `json:"pulses"`
}

// client holds all runtime state for the poll loop. There is one per
// process, created by new_client, then passed by pointer.
type client struct {
	target        string
	state_dir     string
	state_file    string
	poll_interval time.Duration
	http          *http.Client
	pulses        []pulse_record
}

// new_client validates config, creates the state directory, and loads any
// pulses from a previous run. target is required: a client that does not
// know where to poll must not start a loop.
func new_client(target, state_dir string, poll_interval time.Duration) (*client, error) {
	if target == "" {
		return nil, fmt.Errorf("target URL is required")
	}
	if poll_interval <= 0 {
		poll_interval = 60 * time.Second
	}
	if state_dir == "" {
		home_dir, err := os.UserHomeDir()
		if err != nil {
			home_dir = os.Getenv("HOME")
		}
		state_dir = filepath.Join(home_dir, ".pythonic-go-heartbeat")
	}

	c := &client{
		target:        target,
		state_dir:     state_dir,
		state_file:    filepath.Join(state_dir, "pulses.json"),
		poll_interval: poll_interval,
		http:          &http.Client{Timeout: 10 * time.Second},
	}
	if err := c.load_state(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return c, nil
}

// load_state reads the JSON file into c.pulses. A missing file is a first
// boot — we create an empty one. A corrupt file is tolerated with a warning:
// re-recording a pulse is better than refusing to start.
func (c *client) load_state() error {
	data, err := os.ReadFile(c.state_file)
	if os.IsNotExist(err) {
		return c.save_state()
	}
	if err != nil {
		return err
	}

	var s persisted_state
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Printf("Warning: %s is corrupted — starting with empty pulse log\n", c.state_file)
		c.pulses = nil
		return nil
	}
	c.pulses = s.Pulses
	return nil
}

// save_state writes c.pulses to disk. Called after every pulse so the
// history is current even if the process dies mid-loop.
func (c *client) save_state() error {
	if err := os.MkdirAll(c.state_dir, 0755); err != nil {
		return err
	}
	s := persisted_state{Target: c.target, Pulses: c.pulses}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.state_file, data, 0644)
}

// should_record is the idempotency gate. It returns one of:
//
//   - "record"            — never succeeded; take a pulse now
//   - "already_succeeded" — a prior pulse returned 2xx; skip
func (c *client) should_record() string {
	for _, p := range c.pulses {
		if p.Status == "success" {
			return "already_succeeded"
		}
	}
	return "record"
}

// take_pulse GETs the target, appends a record, and flushes to disk.
// Network errors are a failed pulse (we did attempt them). The caller
// decides whether to loop.
func (c *client) take_pulse() pulse_record {
	rec := pulse_record{
		Checked_at: time.Now().UTC().Format(time.RFC3339),
		Status:     "failed",
	}

	resp, err := c.http.Get(c.target)
	if err != nil {
		fmt.Printf("Warning: GET %s: %v\n", c.target, err)
	} else {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		rec.Status_code = resp.StatusCode
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			rec.Status = "success"
		}
	}

	c.pulses = append(c.pulses, rec)
	if err := c.save_state(); err != nil {
		fmt.Printf("Warning: failed to save state: %v\n", err)
	}
	return rec
}

// poll_once runs a single decision-and-pulse cycle. It is the --once path
// and the body of run().
func (c *client) poll_once() pulse_record {
	switch c.should_record() {
	case "already_succeeded":
		fmt.Printf("[%s] already succeeded, skipping\n", time.Now().Format("15:04:05"))
		if len(c.pulses) == 0 {
			return pulse_record{Status: "already_succeeded"}
		}
		return c.pulses[len(c.pulses)-1]
	default:
		rec := c.take_pulse()
		fmt.Printf("[%s] %s status=%d\n", time.Now().Format("15:04:05"), rec.Status, rec.Status_code)
		return rec
	}
}

// run polls forever. Sleep is after the first pulse so a new process does
// useful work immediately. No jitter — this example is a teaching loop,
// not a fleet-wide poller.
func (c *client) run() {
	for {
		c.poll_once()
		time.Sleep(c.poll_interval)
	}
}
