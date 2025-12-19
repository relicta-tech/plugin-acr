package main

import (
	"testing"
)

func TestNewACRClient(t *testing.T) {
	tests := []struct {
		name     string
		registry string
	}{
		{
			name:     "simple registry name",
			registry: "myregistry",
		},
		{
			name:     "full registry url",
			registry: "myregistry.azurecr.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewACRClient(tt.registry)
			if client == nil {
				t.Fatal("expected non-nil client")
			}
		})
	}
}

func TestACRClient_GetRegistryURL(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		expected string
	}{
		{
			name:     "simple name",
			registry: "myregistry",
			expected: "myregistry.azurecr.io",
		},
		{
			name:     "already has suffix",
			registry: "myregistry.azurecr.io",
			expected: "myregistry.azurecr.io",
		},
		{
			name:     "different registry name",
			registry: "contoso",
			expected: "contoso.azurecr.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewACRClient(tt.registry)
			result := client.GetRegistryURL()

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAuthConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *AuthConfig
	}{
		{
			name:   "azure_cli method",
			config: &AuthConfig{Method: "azure_cli"},
		},
		{
			name: "service_principal method",
			config: &AuthConfig{
				Method:       "service_principal",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				TenantID:     "tenant-id",
			},
		},
		{
			name: "admin method",
			config: &AuthConfig{
				Method:   "admin",
				Username: "admin",
				Password: "password",
			},
		},
		{
			name:   "managed_identity method",
			config: &AuthConfig{Method: "managed_identity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config == nil {
				t.Fatal("config should not be nil")
			}
			if tt.config.Method == "" {
				t.Error("method should not be empty")
			}
		})
	}
}
