package main

import (
	"testing"
)

func TestNewDockerClient(t *testing.T) {
	client := NewDockerClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestDockerClient_Methods(t *testing.T) {
	client := NewDockerClient()

	// Test that methods exist and have correct signatures
	// We can't actually call them without Docker, but we can verify the interface

	t.Run("Tag method exists", func(t *testing.T) {
		// Verify the method signature by attempting to get a reference
		_ = client.Tag
	})

	t.Run("Push method exists", func(t *testing.T) {
		// Verify the method signature by attempting to get a reference
		_ = client.Push
	})

	t.Run("ImageExists method exists", func(t *testing.T) {
		// Verify the method signature by attempting to get a reference
		_ = client.ImageExists
	})
}
