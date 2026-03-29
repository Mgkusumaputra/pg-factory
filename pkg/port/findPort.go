package port

// FindFreePort returns the first port >= startPort that is not in use
// locally or by any running Docker container.
func FindFreePort(startPort int) int {
	// Snapshot Docker's occupied ports once — avoids spawning docker ps
	// in a tight loop for every candidate port.
	dockerOccupied := occupiedDockerPorts()

	for p := startPort; ; p++ {
		if !checkLocalPort(p) && !dockerOccupied[p] {
			return p
		}
	}
}
