package port

// FindFreePort returns the first port >= startPort that is not in use
// locally (via netstat) or by any Docker container.
func FindFreePort(startPort int) int {
	port := startPort
	for {
		inUse := false

		// Check if the port is used locally
		if checkLocalPort(port) {
			inUse = true
		}

		// Check if the port is used by Docker
		if checkDockerPort(port) {
			inUse = true
		}

		if !inUse {
			return port
		}
		port++
	}
}
