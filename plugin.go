package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// Version is set at build time.
var Version = "dev"

// ACRPlugin implements the Relicta plugin interface for Azure Container Registry.
type ACRPlugin struct{}

// Config holds the plugin configuration.
type Config struct {
	// ACR Configuration
	Registry   string
	Repository string
	Image      string

	// Authentication
	AuthMethod   string
	ClientID     string
	ClientSecret string
	TenantID     string
	Username     string
	Password     string

	// Source image
	SourceImage string

	// Tags
	Tags []string

	// Behavior
	DryRun bool
}

// GetInfo returns plugin metadata.
func (p *ACRPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "acr",
		Version:     Version,
		Description: "Push container images to Azure Container Registry (ACR)",
		Hooks: []plugin.Hook{
			plugin.HookPostPublish,
		},
	}
}

// Validate validates the plugin configuration.
func (p *ACRPlugin) Validate(ctx context.Context, config map[string]any) (*plugin.ValidateResponse, error) {
	vb := helpers.NewValidationBuilder()
	cfg := p.parseConfig(config)

	// Registry is required
	if cfg.Registry == "" {
		vb.AddError("registry", "ACR registry name is required")
	}

	// Image name is required
	if cfg.Image == "" {
		vb.AddError("image", "image name is required")
	}

	// Source image is required
	if cfg.SourceImage == "" {
		vb.AddError("source_image", "source image is required")
	}

	// Validate auth method
	validMethods := []string{"azure_cli", "service_principal", "admin", "managed_identity", ""}
	isValidMethod := false
	for _, m := range validMethods {
		if cfg.AuthMethod == m {
			isValidMethod = true
			break
		}
	}
	if !isValidMethod {
		vb.AddError("auth.method", "auth method must be 'azure_cli', 'service_principal', 'admin', or 'managed_identity'")
	}

	// Service principal requires credentials
	if cfg.AuthMethod == "service_principal" {
		if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TenantID == "" {
			vb.AddError("auth", "service principal requires client_id, client_secret, and tenant_id")
		}
	}

	// Admin requires credentials
	if cfg.AuthMethod == "admin" {
		if cfg.Username == "" || cfg.Password == "" {
			vb.AddError("auth", "admin auth requires username and password")
		}
	}

	return vb.Build(), nil
}

// Execute runs the plugin logic.
func (p *ACRPlugin) Execute(ctx context.Context, req plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	cfg := p.parseConfig(req.Config)
	cfg.DryRun = cfg.DryRun || req.DryRun

	// Process tag templates
	tags := p.processTags(cfg.Tags, &req.Context)

	// Create ACR client
	client := NewACRClient(cfg.Registry)

	// Authenticate with ACR
	if !cfg.DryRun {
		authCfg := &AuthConfig{
			Method:       cfg.AuthMethod,
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TenantID:     cfg.TenantID,
			Username:     cfg.Username,
			Password:     cfg.Password,
		}
		if err := client.Authenticate(ctx, authCfg); err != nil {
			return nil, fmt.Errorf("failed to authenticate with ACR: %w", err)
		}
	}

	// Create Docker client
	docker := NewDockerClient()

	// Push images
	pushedImages := []string{}
	registryURL := client.GetRegistryURL()

	for _, tag := range tags {
		if tag == "" {
			continue
		}

		// Build image path
		imagePath := cfg.Image
		if cfg.Repository != "" {
			imagePath = fmt.Sprintf("%s/%s", cfg.Repository, cfg.Image)
		}

		targetImage := fmt.Sprintf("%s/%s:%s", registryURL, imagePath, tag)

		if cfg.DryRun {
			fmt.Printf("[dry-run] Would tag %s as %s\n", cfg.SourceImage, targetImage)
			fmt.Printf("[dry-run] Would push %s\n", targetImage)
		} else {
			// Tag the image
			if err := docker.Tag(ctx, cfg.SourceImage, targetImage); err != nil {
				return nil, fmt.Errorf("failed to tag image: %w", err)
			}

			// Push the image
			if err := docker.Push(ctx, targetImage); err != nil {
				return nil, fmt.Errorf("failed to push image: %w", err)
			}

			fmt.Printf("Pushed: %s\n", targetImage)
		}

		pushedImages = append(pushedImages, targetImage)
	}

	return &plugin.ExecuteResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully pushed %d image(s) to ACR", len(pushedImages)),
		Outputs: map[string]any{
			"registry":      registryURL,
			"repository":    cfg.Repository,
			"tags":          tags,
			"pushed_images": pushedImages,
		},
	}, nil
}

// parseConfig parses the raw configuration into a Config struct.
func (p *ACRPlugin) parseConfig(raw map[string]any) *Config {
	parser := helpers.NewConfigParser(raw)

	tags := parser.GetStringSlice("tags", nil)
	if len(tags) == 0 {
		tags = []string{"{{.Version}}"}
	}

	// Parse nested auth config
	authMethod := "azure_cli"
	clientID := ""
	clientSecret := ""
	tenantID := ""
	username := ""
	password := ""
	if authRaw, ok := raw["auth"].(map[string]any); ok {
		authParser := helpers.NewConfigParser(authRaw)
		authMethod = authParser.GetString("method", "", "azure_cli")
		clientID = authParser.GetString("client_id", "AZURE_CLIENT_ID", "")
		clientSecret = authParser.GetString("client_secret", "AZURE_CLIENT_SECRET", "")
		tenantID = authParser.GetString("tenant_id", "AZURE_TENANT_ID", "")
		username = authParser.GetString("username", "ACR_USERNAME", "")
		password = authParser.GetString("password", "ACR_PASSWORD", "")
	}

	return &Config{
		// ACR Configuration
		Registry:   parser.GetString("registry", "", ""),
		Repository: parser.GetString("repository", "", ""),
		Image:      parser.GetString("image", "", ""),

		// Authentication
		AuthMethod:   authMethod,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TenantID:     tenantID,
		Username:     username,
		Password:     password,

		// Source image
		SourceImage: parser.GetString("source_image", "", ""),

		// Tags
		Tags: tags,

		// Behavior
		DryRun: parser.GetBool("dry_run", false),
	}
}

// processTags processes tag templates with release context.
func (p *ACRPlugin) processTags(tags []string, ctx *plugin.ReleaseContext) []string {
	processed := make([]string, 0, len(tags))

	for _, tag := range tags {
		result := p.processTemplate(tag, ctx)
		if result != "" {
			processed = append(processed, result)
		}
	}

	return processed
}

// processTemplate replaces template variables with actual values.
func (p *ACRPlugin) processTemplate(tmpl string, ctx *plugin.ReleaseContext) string {
	result := tmpl

	// Handle conditional templates (simplified)
	if strings.Contains(result, "{{if") {
		return ""
	}

	// Replace common template variables
	result = strings.ReplaceAll(result, "{{.Version}}", ctx.Version)
	result = strings.ReplaceAll(result, "{{.PreviousVersion}}", ctx.PreviousVersion)
	result = strings.ReplaceAll(result, "{{.TagName}}", ctx.TagName)
	result = strings.ReplaceAll(result, "{{.ReleaseType}}", ctx.ReleaseType)

	// Handle branch name
	if ctx.Branch != "" {
		safeBranch := strings.ReplaceAll(ctx.Branch, "/", "-")
		result = strings.ReplaceAll(result, "{{.Branch}}", safeBranch)
	}

	return result
}
