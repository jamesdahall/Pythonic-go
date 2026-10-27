// heartbeat is a tiny poller written in Pythonic Go. It GETs a URL, records
// the result as snake_case JSON, and will not re-record a success.
//
//	go run ./cmd/heartbeat -once https://example.com
package main

import (
	"os"

	"github.com/jamesdahall/Pythonic-go/examples/heartbeat"
)

func main() {
	os.Exit(heartbeat.Run(os.Args[1:]))
}
