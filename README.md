# mage-common

[![CI](https://github.com/randomvariable/mage-common/actions/workflows/ci.yml/badge.svg)](https://github.com/randomvariable/mage-common/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/randomvariable/mage-common.svg)](https://pkg.go.dev/github.com/randomvariable/mage-common)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Common utilities and libraries for [Mage](https://magefile.org/) build automation.

## Packages

### config

Standardized Viper-based configuration loading for Mage projects with support for:
- Graceful `config.yaml` loading with automatic fallback
- Environment variable overrides
- Named command-line arguments (`--key=value`)
- Mage integration via `CleanOSArgs()` to prevent flag conflicts
- Structured logging with sensitive value redaction

**[Full documentation →](config/README.md)**

**Quick Example:**

```go
//go:build mage

package main

import (
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func init() {
    pflag.String("image", "", "Container image name")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()
}

// Usage: mage build --image=webapp --tag=v1.0
func Build() error {
    image := viper.GetString("image")
    tag := viper.GetString("tag")
    // ... build logic
}
```

## Installation

```bash
go get github.com/randomvariable/mage-common/config
```

## Requirements

- Go 1.23 or later
- [Mage](https://magefile.org/) (for magefiles)

## Contributing

Contributions are welcome! Please ensure:
- All tests pass: `go test ./...`
- Code is linted: `golangci-lint run`
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
