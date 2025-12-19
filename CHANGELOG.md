# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2024-12-19

### Added

- Initial release
- Push container images to Azure Container Registry
- Multiple authentication methods:
  - Azure CLI (default)
  - Service Principal
  - Admin credentials
  - Managed Identity
- Dynamic tag templating with release context variables
- Repository organization support
- Dry-run mode for testing configurations
- Comprehensive validation of configuration
