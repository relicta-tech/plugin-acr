package main

import (
	"context"
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

func TestACRPlugin_GetInfo(t *testing.T) {
	p := &ACRPlugin{}
	info := p.GetInfo()

	if info.Name != "acr" {
		t.Errorf("expected name 'acr', got %q", info.Name)
	}

	if info.Description == "" {
		t.Error("expected non-empty description")
	}

	if len(info.Hooks) == 0 {
		t.Error("expected at least one hook")
	}

	hasPostPublish := false
	for _, h := range info.Hooks {
		if h == plugin.HookPostPublish {
			hasPostPublish = true
			break
		}
	}
	if !hasPostPublish {
		t.Error("expected HookPostPublish in hooks")
	}
}

func TestACRPlugin_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]any
		wantErrors  int
		description string
	}{
		{
			name:        "valid config with azure_cli",
			config:      map[string]any{"registry": "myregistry", "image": "myapp", "source_image": "myapp:latest"},
			wantErrors:  0,
			description: "should pass with minimal required fields",
		},
		{
			name:        "missing registry",
			config:      map[string]any{"image": "myapp", "source_image": "myapp:latest"},
			wantErrors:  1,
			description: "should fail when registry is missing",
		},
		{
			name:        "missing image",
			config:      map[string]any{"registry": "myregistry", "source_image": "myapp:latest"},
			wantErrors:  1,
			description: "should fail when image is missing",
		},
		{
			name:        "missing source_image",
			config:      map[string]any{"registry": "myregistry", "image": "myapp"},
			wantErrors:  1,
			description: "should fail when source_image is missing",
		},
		{
			name:        "invalid auth method",
			config:      map[string]any{"registry": "myregistry", "image": "myapp", "source_image": "myapp:latest", "auth": map[string]any{"method": "invalid"}},
			wantErrors:  1,
			description: "should fail with invalid auth method",
		},
		{
			name: "service_principal missing credentials",
			config: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth":         map[string]any{"method": "service_principal"},
			},
			wantErrors:  1,
			description: "should fail when service principal credentials are missing",
		},
		{
			name: "service_principal with credentials",
			config: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth": map[string]any{
					"method":        "service_principal",
					"client_id":     "my-client-id",
					"client_secret": "my-secret",
					"tenant_id":     "my-tenant",
				},
			},
			wantErrors:  0,
			description: "should pass with service principal credentials",
		},
		{
			name: "admin missing credentials",
			config: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth":         map[string]any{"method": "admin"},
			},
			wantErrors:  1,
			description: "should fail when admin credentials are missing",
		},
		{
			name: "admin with credentials",
			config: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth": map[string]any{
					"method":   "admin",
					"username": "admin",
					"password": "password123",
				},
			},
			wantErrors:  0,
			description: "should pass with admin credentials",
		},
		{
			name: "managed_identity auth",
			config: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth":         map[string]any{"method": "managed_identity"},
			},
			wantErrors:  0,
			description: "should pass with managed identity auth",
		},
		{
			name:        "empty config",
			config:      map[string]any{},
			wantErrors:  3,
			description: "should fail with multiple errors for empty config",
		},
	}

	p := &ACRPlugin{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Validate(context.Background(), tt.config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resp.Errors) != tt.wantErrors {
				t.Errorf("%s: expected %d errors, got %d: %v",
					tt.description, tt.wantErrors, len(resp.Errors), resp.Errors)
			}
		})
	}
}

func TestACRPlugin_Execute_DryRun(t *testing.T) {
	p := &ACRPlugin{}

	req := plugin.ExecuteRequest{
		Hook:   plugin.HookPostPublish,
		DryRun: true,
		Config: map[string]any{
			"registry":     "myregistry",
			"image":        "myapp",
			"source_image": "myapp:latest",
			"tags":         []any{"1.0.0", "latest"},
		},
		Context: plugin.ReleaseContext{
			Version: "1.0.0",
		},
	}

	resp, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected success in dry run")
	}

	pushedImages, ok := resp.Outputs["pushed_images"].([]string)
	if !ok {
		t.Fatal("expected pushed_images in outputs")
	}

	if len(pushedImages) != 2 {
		t.Errorf("expected 2 pushed images, got %d", len(pushedImages))
	}
}

func TestACRPlugin_ProcessTags(t *testing.T) {
	p := &ACRPlugin{}

	tests := []struct {
		name     string
		tags     []string
		ctx      *plugin.ReleaseContext
		expected []string
	}{
		{
			name: "version template",
			tags: []string{"{{.Version}}"},
			ctx:  &plugin.ReleaseContext{Version: "1.0.0"},
			expected: []string{"1.0.0"},
		},
		{
			name: "multiple templates",
			tags: []string{"{{.Version}}", "latest", "{{.TagName}}"},
			ctx:  &plugin.ReleaseContext{Version: "1.0.0", TagName: "v1.0.0"},
			expected: []string{"1.0.0", "latest", "v1.0.0"},
		},
		{
			name: "branch template with slashes",
			tags: []string{"{{.Branch}}"},
			ctx:  &plugin.ReleaseContext{Branch: "feature/my-feature"},
			expected: []string{"feature-my-feature"},
		},
		{
			name:     "conditional template skipped",
			tags:     []string{"{{if .IsPrerelease}}beta{{end}}"},
			ctx:      &plugin.ReleaseContext{Version: "1.0.0"},
			expected: []string{},
		},
		{
			name: "release type template",
			tags: []string{"{{.ReleaseType}}"},
			ctx:  &plugin.ReleaseContext{ReleaseType: "stable"},
			expected: []string{"stable"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.processTags(tt.tags, tt.ctx)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d tags, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, tag := range result {
				if tag != tt.expected[i] {
					t.Errorf("tag %d: expected %q, got %q", i, tt.expected[i], tag)
				}
			}
		})
	}
}

func TestACRPlugin_ParseConfig(t *testing.T) {
	p := &ACRPlugin{}

	tests := []struct {
		name     string
		raw      map[string]any
		check    func(*Config) error
	}{
		{
			name: "basic config",
			raw: map[string]any{
				"registry":     "myregistry",
				"repository":   "myrepo",
				"image":        "myapp",
				"source_image": "myapp:latest",
			},
			check: func(c *Config) error {
				if c.Registry != "myregistry" {
					return errorf("expected registry 'myregistry', got %q", c.Registry)
				}
				if c.Repository != "myrepo" {
					return errorf("expected repository 'myrepo', got %q", c.Repository)
				}
				if c.Image != "myapp" {
					return errorf("expected image 'myapp', got %q", c.Image)
				}
				if c.SourceImage != "myapp:latest" {
					return errorf("expected source_image 'myapp:latest', got %q", c.SourceImage)
				}
				return nil
			},
		},
		{
			name: "auth config",
			raw: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"auth": map[string]any{
					"method":        "service_principal",
					"client_id":     "my-client",
					"client_secret": "my-secret",
					"tenant_id":     "my-tenant",
				},
			},
			check: func(c *Config) error {
				if c.AuthMethod != "service_principal" {
					return errorf("expected auth method 'service_principal', got %q", c.AuthMethod)
				}
				if c.ClientID != "my-client" {
					return errorf("expected client_id 'my-client', got %q", c.ClientID)
				}
				if c.ClientSecret != "my-secret" {
					return errorf("expected client_secret 'my-secret', got %q", c.ClientSecret)
				}
				if c.TenantID != "my-tenant" {
					return errorf("expected tenant_id 'my-tenant', got %q", c.TenantID)
				}
				return nil
			},
		},
		{
			name: "default tags",
			raw: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
			},
			check: func(c *Config) error {
				if len(c.Tags) != 1 || c.Tags[0] != "{{.Version}}" {
					return errorf("expected default tag [{{.Version}}], got %v", c.Tags)
				}
				return nil
			},
		},
		{
			name: "custom tags",
			raw: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"tags":         []any{"v1.0.0", "latest"},
			},
			check: func(c *Config) error {
				if len(c.Tags) != 2 {
					return errorf("expected 2 tags, got %d", len(c.Tags))
				}
				if c.Tags[0] != "v1.0.0" || c.Tags[1] != "latest" {
					return errorf("expected tags [v1.0.0, latest], got %v", c.Tags)
				}
				return nil
			},
		},
		{
			name: "dry_run config",
			raw: map[string]any{
				"registry":     "myregistry",
				"image":        "myapp",
				"source_image": "myapp:latest",
				"dry_run":      true,
			},
			check: func(c *Config) error {
				if !c.DryRun {
					return errorf("expected dry_run to be true")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := p.parseConfig(tt.raw)
			if err := tt.check(cfg); err != nil {
				t.Error(err)
			}
		})
	}
}

// errorf is a helper to create formatted errors.
func errorf(format string, args ...any) error {
	return &testError{msg: format, args: args}
}

type testError struct {
	msg  string
	args []any
}

func (e *testError) Error() string {
	return e.msg
}
