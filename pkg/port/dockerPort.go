package port

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// hostPortRe matches the host-side port in Docker's Ports format,
// e.g. "0.0.0.0:5432->5432/tcp" → captures "5432".
var hostPortRe = regexp.MustCompile(`(?:^|,\s*)(?:\S+:)?(\d+)->`)

// occupiedDockerPorts returns the set of host ports currently bound by running
// Docker containers. Called once before the FindFreePort loop to avoid
// spawning docker ps on every candidate port.
func occupiedDockerPorts() map[int]bool {
	cmd := exec.Command("docker", "ps", "--format", "{{.Ports}}")
	out, err := cmd.Output()
	if err != nil {
		return map[int]bool{}
	}

	occupied := make(map[int]bool)
	for _, line := range strings.Split(string(out), "\n") {
		for _, m := range hostPortRe.FindAllStringSubmatch(line, -1) {
			if p, err := strconv.Atoi(m[1]); err == nil {
				occupied[p] = true
			}
		}
	}
	return occupied
}
