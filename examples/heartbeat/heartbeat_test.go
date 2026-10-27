package heartbeat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Test_new_client_requires_target(t *testing.T) {
	_, err := new_client("", t.TempDir(), time.Second)
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}

func Test_should_record_empty_history(t *testing.T) {
	c := &client{}
	if got := c.should_record(); got != "record" {
		t.Fatalf("got %q, want %q", got, "record")
	}
}

func Test_should_record_skips_after_success(t *testing.T) {
	c := &client{
		pulses: []pulse_record{
			{Status: "failed", Status_code: 500},
			{Status: "success", Status_code: 200},
		},
	}
	if got := c.should_record(); got != "already_succeeded" {
		t.Fatalf("got %q, want %q", got, "already_succeeded")
	}
}

func Test_should_record_retries_after_only_failures(t *testing.T) {
	c := &client{
		pulses: []pulse_record{
			{Status: "failed", Status_code: 503},
		},
	}
	if got := c.should_record(); got != "record" {
		t.Fatalf("got %q, want %q", got, "record")
	}
}

func Test_take_pulse_records_success_and_persists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	c, err := new_client(server.URL, dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	rec := c.take_pulse()
	if rec.Status != "success" {
		t.Fatalf("status=%q code=%d, want success", rec.Status, rec.Status_code)
	}

	data, err := os.ReadFile(filepath.Join(dir, "pulses.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s persisted_state
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Pulses) != 1 || s.Pulses[0].Status != "success" {
		t.Fatalf("persisted pulses = %+v", s.Pulses)
	}
}

func Test_poll_once_skips_when_already_succeeded(t *testing.T) {
	// A server that panics if it is hit — skipping must not call it.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server was called after a recorded success")
	}))
	defer server.Close()

	c, err := new_client(server.URL, t.TempDir(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c.pulses = []pulse_record{{Status: "success", Status_code: 200}}

	rec := c.poll_once()
	if rec.Status != "success" {
		t.Fatalf("got %+v", rec)
	}
}

func Test_load_state_tolerates_corrupt_file(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pulses.json")
	if err := os.WriteFile(path, []byte("not-json"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := new_client("http://127.0.0.1:1", dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if c.pulses != nil {
		t.Fatalf("expected empty pulses after corrupt file, got %+v", c.pulses)
	}
}
