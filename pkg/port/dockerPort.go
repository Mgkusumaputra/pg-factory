package port

import (
	"fmt"
	"os/exec"
	"strings"
)

func checkDockerPort(port int) bool {
	cmd := exec.Command("docker", "ps", "--format", "{{.Ports}}")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(out), "\n")
	portStr := fmt.Sprintf(":%d->", port)
	for _, line := range lines {
		if strings.Contains(line, portStr) {
			return true
		}
	}
	return false
}
