package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type DockerService struct {
	Timeout time.Duration
}

func NewDockerService(timeout time.Duration) *DockerService {
	return &DockerService{Timeout: timeout}
}

// RunCommand executes a docker command and returns stdout and stderr.
func (d *DockerService) RunCommand(args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ContainerExists returns true if a container with the given name exists (running or stopped).
func (d *DockerService) ContainerExists(name string) (bool, error) {
	stdout, _, err := d.RunCommand("ps", "-a", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(stdout) == name, nil
}

// ContainerRunning returns true if the container is currently running.
func (d *DockerService) ContainerRunning(name string) (bool, error) {
	stdout, _, err := d.RunCommand("ps", "--filter", "name=^"+name+"$", "--format", "{{.Names}}")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(stdout) == name, nil
}

// StartContainer starts a stopped container by name.
func (d *DockerService) StartContainer(name string) error {
	_, stderr, err := d.RunCommand("start", name)
	if err != nil {
		return fmt.Errorf("docker start failed: %s", stderr)
	}
	return nil
}

// StopContainer stops a running container by name.
func (d *DockerService) StopContainer(name string) error {
	_, stderr, err := d.RunCommand("stop", name)
	if err != nil {
		return fmt.Errorf("docker stop failed: %s", stderr)
	}
	return nil
}

// RemoveContainer removes a container by name (must be stopped first).
func (d *DockerService) RemoveContainer(name string) error {
	_, stderr, err := d.RunCommand("rm", name)
	if err != nil {
		return fmt.Errorf("docker rm failed: %s", stderr)
	}
	return nil
}

// RemoveVolume removes a Docker volume by name.
func (d *DockerService) RemoveVolume(name string) error {
	_, stderr, err := d.RunCommand("volume", "rm", name)
	if err != nil {
		return fmt.Errorf("docker volume rm failed: %s", stderr)
	}
	return nil
}

// RunPostgres creates and starts a new Postgres container.
func (d *DockerService) RunPostgres(containerName, volumeName, version, user, password, db string, port int) error {
	args := []string{
		"run", "-d",
		"--name", containerName,
		"-e", fmt.Sprintf("POSTGRES_USER=%s", user),
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", db),
		"-p", fmt.Sprintf("%d:5432", port),
		"-v", fmt.Sprintf("%s:/var/lib/postgresql/data", volumeName),
		fmt.Sprintf("postgres:%s", version),
	}
	_, stderr, err := d.RunCommand(args...)
	if err != nil {
		return fmt.Errorf("docker run failed: %s", stderr)
	}
	return nil
}

// RunningContainerNames returns a set of container names that are currently
// running. Makes a single docker ps call regardless of instance count.
func (d *DockerService) RunningContainerNames() (map[string]bool, error) {
	stdout, _, err := d.RunCommand("ps", "--format", "{{.Names}}")
	if err != nil {
		return nil, err
	}
	running := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		name := strings.TrimSpace(line)
		if name != "" {
			running[name] = true
		}
	}
	return running, nil
}

// RenameContainer renames a Docker container. Works whether the container is
// running or stopped.
func (d *DockerService) RenameContainer(oldName, newName string) error {
	_, stderr, err := d.RunCommand("rename", oldName, newName)
	if err != nil {
		return fmt.Errorf("docker rename failed: %s", stderr)
	}
	return nil
}