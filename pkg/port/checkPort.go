package port

import (
	"fmt"
	"net"
)

// checkLocalPort returns true when the given port is already bound on the
// local machine. Uses a pure-Go net.Listen probe — no external binary needed,
// works identically on Windows, macOS, and Linux.
//
// We attempt to listen on the port: if that succeeds, the port is free; if it
// fails (EADDRINUSE or similar), the port is in use. The Dial approach was
// removed because after a successful Listen+Close the port is immediately
// available again, making a subsequent Dial always fail — a no-op that added
// 200 ms per candidate port.
func checkLocalPort(p int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		return true // couldn't bind → port is in use
	}
	l.Close()
	return false
}
