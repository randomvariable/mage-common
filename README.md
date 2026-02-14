# mage-common

[![CI](https://github.com/randomvariable/mage-common/actions/workflows/ci.yml/badge.svg)](https://github.com/randomvariable/mage-common/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/randomvariable/mage-common.svg)](https://pkg.go.dev/github.com/randomvariable/mage-common)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Common utilities and libraries for [Mage](https://magefile.org/) build automation.

## Packages

### tools

Declarative tool dependency management for Mage projects. Define your tool dependencies in `.tools.yaml` and let the framework handle installation, version tracking, and execution across Go, download, cargo, gem, npx, and uvx source types.

**[Full documentation →](tools/README.md)**

**Quick Example:**

```yaml
# .tools.yaml
apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  toolsDir: hack/bin
  tools:
    - name: golangci-lint
      version: v2.9.0
      sources:
        - type: golangci-lint
          url: github.com/golangci/golangci-lint/v2/cmd/golangci-lint

    - name: kubectl
      version: v1.33.1
      sources:
        - type: download
          url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl"
          checksum:
            url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl.sha256"
```

```go
//go:build mage

package main

import (
    "context"

    "github.com/spf13/pflag"

    "github.com/randomvariable/mage-common/config"
    //mage:import tools
    _ "github.com/randomvariable/mage-common/tools/targets"

    magetools "github.com/randomvariable/mage-common/tools"
)

func init() {
    pflag.Parse()
    config.CleanOSArgs()
}

// Lint runs golangci-lint (auto-installed from .tools.yaml).
func Lint(ctx context.Context) error {
    _, err := magetools.Run(ctx, "golangci-lint", []string{"run", "./..."})
    return err
}
```

### config

Standardized Viper-based configuration loading for Mage projects with support for:
- Graceful `config.yaml` loading with automatic fallback
- Environment variable overrides
- Named command-line arguments (`--key=value`)
- Mage integration via `CleanOSArgs()` to prevent flag conflicts
- Structured logging with sensitive value redaction

**[Full documentation →](config/README.md)**

## Installation

```bash
go get github.com/randomvariable/mage-common/tools
go get github.com/randomvariable/mage-common/config
```

## Requirements

- Go 1.25 or later
- [Mage](https://magefile.org/) (for magefiles)

## Contributing

Contributions are welcome! Please ensure:
- All tests pass: `mage test:run` or `go test ./...`
- Code is linted: `mage lint:run` or `golangci-lint run`
- Add tests for new functionality

## License

Licensed under the [Apache License, Version 2.0](LICENSE).

Copyright 2026 Naadir Jeewa

## Versioning

This project follows [Semantic Versioning](https://semver.org/).

## Support

For issues and questions:
- [GitHub Issues](https://github.com/randomvariable/mage-common/issues)
- [Discussions](https://github.com/randomvariable/mage-common/discussions)
