package main

import (
	"context"
	"fmt"
	"os/exec"
)

// DockerClient provides Docker CLI operations.
type DockerClient struct{}

// NewDockerClient creates a new Docker client.
func NewDockerClient() *DockerClient {
	return &DockerClient{}
}

// Tag tags a Docker image.
func (d *DockerClient) Tag(ctx context.Context, source, target string) error {
	cmd := exec.CommandContext(ctx, "docker", "tag", source, target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker tag failed: %w\n%s", err, string(output))
	}
	return nil
}

// Push pushes a Docker image.
func (d *DockerClient) Push(ctx context.Context, image string) error {
	cmd := exec.CommandContext(ctx, "docker", "push", image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w\n%s", err, string(output))
	}
	return nil
}

// ImageExists checks if a Docker image exists locally.
func (d *DockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}
