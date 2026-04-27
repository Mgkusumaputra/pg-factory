package port

import "fmt"

// FindFreePort returns the first port >= startPort that is not in use
// locally or by any running Docker container.
func FindFreePort(startPort int) (int, error) {
	if startPort < 1024 || startPort > 65535 {
		return 0, fmt.Errorf("invalid start port %d: must be between 1024 and 65535", startPort)
	}

	// Snapshot Docker's occupied ports once — avoids spawning docker ps
	// in a tight loop for every candidate port.
	dockerOccupied := occupiedDockerPorts()

	for p := startPort; p <= 65535; p++ {
		if !checkLocalPort(p) && !dockerOccupied[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port available in range %d-65535", startPort)
}
