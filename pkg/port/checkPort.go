package port

import (
	"fmt"
	"net"
	"time"
)

// checkLocalPort returns true when the given port is already bound on the
// local machine. Uses a pure-Go net.Listen probe — no external binary needed,
// works identically on Windows, macOS, and Linux.
func checkLocalPort(p int) bool {
	// First try binding: if Listen succeeds the port is free.
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		return true // couldn't bind → port is in use
	}
	l.Close()

	// Also try dialing to catch ports in TIME_WAIT or held by another process.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", p), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return true // something answered → port is in use
	}
	return false
}
