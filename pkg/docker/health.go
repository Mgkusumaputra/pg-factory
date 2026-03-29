// pkg/docker/health.go
// Readiness polling for Postgres containers.
package docker

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// WaitUntilReady polls `docker exec <container> pg_isready -U <user>` every
// second until Postgres accepts connections or the timeout is reached.
func (d *DockerService) WaitUntilReady(containerName, user string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isReady(containerName, user) {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timed out waiting for Postgres in %q to become ready", containerName)
}

// isReady runs pg_isready inside the container and returns true on exit code 0.
func isReady(containerName, user string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "exec", containerName,
		"pg_isready", "-U", user)
	return cmd.Run() == nil
}
