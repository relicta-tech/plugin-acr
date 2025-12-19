package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Method       string
	ClientID     string
	ClientSecret string
	TenantID     string
	Username     string
	Password     string
}

// ACRClient provides ACR operations.
type ACRClient struct {
	registry string
}

// NewACRClient creates a new ACR client.
func NewACRClient(registry string) *ACRClient {
	return &ACRClient{
		registry: registry,
	}
}

// Authenticate authenticates with ACR.
func (c *ACRClient) Authenticate(ctx context.Context, auth *AuthConfig) error {
	if auth == nil {
		auth = &AuthConfig{Method: "azure_cli"}
	}

	switch auth.Method {
	case "azure_cli", "":
		return c.authenticateAzureCLI(ctx)
	case "service_principal":
		return c.authenticateServicePrincipal(ctx, auth)
	case "admin":
		return c.authenticateAdmin(ctx, auth)
	case "managed_identity":
		return c.authenticateManagedIdentity(ctx)
	default:
		return fmt.Errorf("unknown auth method: %s", auth.Method)
	}
}

// authenticateAzureCLI uses Azure CLI for authentication.
func (c *ACRClient) authenticateAzureCLI(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "az", "acr", "login", "--name", c.registry)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("az acr login failed: %w\n%s", err, string(output))
	}
	return nil
}

// authenticateServicePrincipal uses service principal for authentication.
func (c *ACRClient) authenticateServicePrincipal(ctx context.Context, auth *AuthConfig) error {
	// Login to Azure first
	loginCmd := exec.CommandContext(ctx, "az", "login",
		"--service-principal",
		"-u", auth.ClientID,
		"-p", auth.ClientSecret,
		"--tenant", auth.TenantID,
	)
	output, err := loginCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("azure login failed: %w\n%s", err, string(output))
	}

	// Then login to ACR
	return c.authenticateAzureCLI(ctx)
}

// authenticateAdmin uses admin credentials for authentication.
func (c *ACRClient) authenticateAdmin(ctx context.Context, auth *AuthConfig) error {
	cmd := exec.CommandContext(ctx, "docker", "login",
		c.GetRegistryURL(),
		"-u", auth.Username,
		"--password-stdin",
	)
	cmd.Stdin = strings.NewReader(auth.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w\n%s", err, string(output))
	}

	return nil
}

// authenticateManagedIdentity uses managed identity for authentication.
func (c *ACRClient) authenticateManagedIdentity(ctx context.Context) error {
	// Use az acr login which automatically uses managed identity
	cmd := exec.CommandContext(ctx, "az", "acr", "login", "--name", c.registry)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("az acr login with managed identity failed: %w\n%s", err, string(output))
	}
	return nil
}

// GetRegistryURL returns the full ACR URL.
func (c *ACRClient) GetRegistryURL() string {
	// If registry already has .azurecr.io, return as-is
	if strings.HasSuffix(c.registry, ".azurecr.io") {
		return c.registry
	}
	return fmt.Sprintf("%s.azurecr.io", c.registry)
}
