package heartbeat

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Run is the CLI entrypoint for the heartbeat example. cmd/heartbeat calls
// this so the package stays importable by tests.
func Run(args []string) int {
	fs := flag.NewFlagSet("heartbeat", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	once := fs.Bool("once", false, "Take one pulse and exit")
	interval := fs.Duration("interval", 60*time.Second, "Delay between pulses")
	state_dir := fs.String("state-dir", "", "Directory for pulses.json (default ~/.pythonic-go-heartbeat)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	target := fs.Arg(0)
	c, err := new_client(target, *state_dir, *interval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Println("Heartbeat started")
	fmt.Printf("  Target:   %s\n", c.target)
	fmt.Printf("  State:    %s\n", c.state_file)
	fmt.Printf("  Interval: %s\n", c.poll_interval)

	if *once {
		c.poll_once()
		return 0
	}
	c.run()
	return 0
}
