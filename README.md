# plugin-acr

A Relicta plugin for pushing container images to Azure Container Registry (ACR).

## Features

- Push Docker images to Azure Container Registry
- Multiple authentication methods:
  - Azure CLI (default)
  - Service Principal
  - Admin credentials
  - Managed Identity
- Dynamic tag templating with release context
- Repository organization support
- Dry-run mode for testing

## Installation

```bash
relicta plugin install acr
```

## Configuration

```yaml
plugins:
  acr:
    # Required: ACR registry name (without .azurecr.io suffix)
    registry: myregistry

    # Required: Image name to push
    image: myapp

    # Required: Source image to tag and push
    source_image: myapp:latest

    # Optional: Repository/namespace within ACR
    repository: myproject

    # Optional: Tags to apply (supports templates)
    tags:
      - "{{.Version}}"
      - latest
      - "{{.Branch}}"

    # Optional: Authentication configuration
    auth:
      # Method: azure_cli (default), service_principal, admin, managed_identity
      method: azure_cli

      # For service_principal method:
      client_id: ${AZURE_CLIENT_ID}
      client_secret: ${AZURE_CLIENT_SECRET}
      tenant_id: ${AZURE_TENANT_ID}

      # For admin method:
      username: ${ACR_USERNAME}
      password: ${ACR_PASSWORD}

    # Optional: Dry run mode
    dry_run: false
```

## Authentication Methods

### Azure CLI (Default)

Uses `az acr login` with the current Azure CLI session. Requires Azure CLI to be installed and logged in.

```yaml
auth:
  method: azure_cli
```

### Service Principal

Uses Azure Service Principal credentials for CI/CD environments.

```yaml
auth:
  method: service_principal
  client_id: ${AZURE_CLIENT_ID}
  client_secret: ${AZURE_CLIENT_SECRET}
  tenant_id: ${AZURE_TENANT_ID}
```

### Admin Credentials

Uses ACR admin username and password. Must be enabled on the registry.

```yaml
auth:
  method: admin
  username: ${ACR_USERNAME}
  password: ${ACR_PASSWORD}
```

### Managed Identity

Uses Azure Managed Identity for Azure-hosted workloads.

```yaml
auth:
  method: managed_identity
```

## Tag Templates

Tags support Go template syntax with access to release context:

| Template | Description |
|----------|-------------|
| `{{.Version}}` | Release version (e.g., `1.0.0`) |
| `{{.PreviousVersion}}` | Previous release version |
| `{{.TagName}}` | Git tag name (e.g., `v1.0.0`) |
| `{{.Branch}}` | Branch name (slashes replaced with dashes) |
| `{{.ReleaseType}}` | Release type (e.g., `stable`, `prerelease`) |

## Outputs

The plugin provides the following outputs:

| Output | Description |
|--------|-------------|
| `registry` | Full registry URL |
| `repository` | Repository name |
| `tags` | List of processed tags |
| `pushed_images` | List of pushed image references |

## Examples

### Basic Usage

```yaml
plugins:
  acr:
    registry: mycompany
    image: myapp
    source_image: myapp:latest
```

### With Repository

```yaml
plugins:
  acr:
    registry: mycompany
    repository: backend
    image: api-server
    source_image: api-server:latest
    tags:
      - "{{.Version}}"
      - latest
```

### CI/CD with Service Principal

```yaml
plugins:
  acr:
    registry: mycompany
    image: myapp
    source_image: myapp:latest
    auth:
      method: service_principal
      client_id: ${AZURE_CLIENT_ID}
      client_secret: ${AZURE_CLIENT_SECRET}
      tenant_id: ${AZURE_TENANT_ID}
    tags:
      - "{{.Version}}"
      - "{{.Branch}}"
```

## Requirements

- Docker CLI installed and running
- Azure CLI (for `azure_cli` and `managed_identity` methods)
- Appropriate Azure permissions for the registry

## License

MIT License - see [LICENSE](LICENSE) for details.
