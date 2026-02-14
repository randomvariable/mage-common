# tools

Declarative tool dependency management for Mage projects.

## Features

- **Declarative Configuration**: Define tool dependencies in `.tools.yaml` with versioned, typed sources
- **Multiple Install Types**: Go, download, golangci-lint, cargo, gem, npx, and uvx
- **Source Fallback**: List multiple sources per tool; the installer tries each in order until one succeeds
- **Version Tracking**: Symlink-based version management with automatic skip when already installed
- **Checksum Verification**: SHA-256/384/512 verification for downloads (inline or URL-based)
- **Archive Extraction**: Automatic extraction of `.tar.gz` and `.zip` archives
- **Platform Awareness**: Binaries installed under `<toolsDir>/<GOOS>/<GOARCH>/`
- **Once-Per-Process Guarantee**: `Ensure` and `Run` install each tool at most once per Mage invocation
- **Concurrent Installation**: Different source types install in parallel; tools within the same type run sequentially
- **Version Updates**: Query upstream registries (Go proxy, npm, crates.io, PyPI, RubyGems) for latest versions
- **CI Cache Keys**: Generate deterministic cache keys from config content and platform
- **Mage Targets**: Import-ready targets for `tools:install`, `tools:ensure`, and `tools:verify`

## Installation

```bash
go get github.com/randomvariable/mage-common/tools
```

## Quick Start

### 1. Create `.tools.yaml`

```yaml
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

    - name: gotestsum
      version: v1.13.0
      sources:
        - type: go
          url: gotest.tools/gotestsum

    - name: kubectl
      version: v1.33.1
      sources:
        - type: download
          url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl"
          checksum:
            url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl.sha256"
```

### 2. Import Mage Targets

```go
//go:build mage

package main

import (
    "github.com/spf13/pflag"

    "github.com/randomvariable/mage-common/config"
    //mage:import tools
    _ "github.com/randomvariable/mage-common/tools/targets"
)

func init() {
    pflag.Parse()
    config.CleanOSArgs()
}
```

This gives you:

```
$ mage -l
Targets:
  tools:ensure    installs a single tool by name from .tools.yaml.
  tools:install   installs all configured tools from .tools.yaml.
  tools:verify    validates the .tools.yaml configuration file.
```

### 3. Use in Targets

```go
import magetools "github.com/randomvariable/mage-common/tools"

func Lint(ctx context.Context) error {
    // Ensure + run: installs golangci-lint if needed, then executes it
    _, err := magetools.Run(ctx, "golangci-lint", []string{"run", "./..."})
    return err
}
```

## Configuration Reference

### Top-Level Fields

| Field | Type | Description |
|-------|------|-------------|
| `apiVersion` | string | Must be `mage-common.randomvariable.co.uk/v1alpha1` |
| `kind` | string | Must be `ToolConfiguration` |
| `spec.toolsDir` | string | Base directory for installed binaries (default: `hack/tools`) |
| `spec.tools` | list | Tool definitions |

### Tool Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Binary name and unique identifier |
| `version` | string | Pinned version (semver tag). Omit for local-only tools |
| `tagPrefix` | string | Git tag prefix for version discovery (e.g. `kustomize/`) |
| `sources` | list | Priority-ordered installation sources |

### Source Types

#### `go` -- Go install

Installs via `go install <url>@<version>` with `CGO_ENABLED=0`.

```yaml
- name: controller-gen
  version: v0.18.0
  sources:
    - type: go
      url: sigs.k8s.io/controller-tools/cmd/controller-gen
```

For local in-tree tools, set `path` instead of `url`:

```yaml
- name: my-tool
  sources:
    - type: go
      path: ./cmd/my-tool
```

#### `download` -- HTTP download

Downloads a binary or archive via HTTP. Supports Go template tags in URLs.

```yaml
- name: kubectl
  version: v1.33.1
  sources:
    - type: download
      url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl"
      checksum:
        url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl.sha256"
```

**Template variables**: `{{.Version}}`, `{{.OS}}`, `{{.Arch}}`

**Archive extraction**: Set `binaryPath` to extract a binary from a `.tar.gz` or `.zip` archive:

```yaml
- name: kustomize
  version: v5.6.0
  tagPrefix: "kustomize/"
  sources:
    - type: download
      url: "https://github.com/.../kustomize_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
      binaryPath: kustomize
      checksum:
        url: "https://github.com/.../checksums.txt"
```

**Checksum verification**: Supports both bare hash files (single hex digest) and GNU coreutils format (`<hash>  <filename>`). Inline checksums use `<algorithm>:<hex>` format:

```yaml
checksum:
  inline: "sha256:abc123..."
```

**Platform mapping**: Use `osMap` and `archMap` to translate `GOOS`/`GOARCH` values:

```yaml
osMap:
  darwin: macOS
archMap:
  amd64: x86_64
```

#### `golangci-lint` -- golangci-lint with custom plugins

Installs golangci-lint via `go install`. If `.custom-gcl.yml` is present, builds a custom binary with module plugins using `golangci-lint custom`.

```yaml
- name: golangci-lint
  version: v2.9.0
  sources:
    - type: golangci-lint
      url: github.com/golangci/golangci-lint/v2/cmd/golangci-lint
```

#### `cargo` -- Rust crate

Installs via `cargo install <package> --version <version>`.

```yaml
- name: typos
  version: v1.32.0
  sources:
    - type: cargo
      package: typos-cli
```

#### `gem` -- Ruby gem

Installs via `gem install <package> -v <version>` into the tools directory.

```yaml
- name: mdl
  version: 0.13.0
  sources:
    - type: gem
      package: mdl
```

#### `npx` -- npm package

Creates a shell shim that delegates to `npx <package>@<version>`.

```yaml
- name: markdownlint-cli2
  version: v0.17.2
  sources:
    - type: npx
      package: markdownlint-cli2
```

#### `uvx` -- Python package

Creates a shell shim that delegates to `uvx <package>==<version>`.

```yaml
- name: yamllint
  version: 1.37.1
  sources:
    - type: uvx
      package: yamllint
```

### Source Fallback

List multiple sources to try in order. This lets you prefer a fast download but fall back to building from source when a pre-built binary is unavailable:

```yaml
- name: my-tool
  version: v1.0.0
  sources:
    - type: download
      url: "https://github.com/.../my-tool_{{.OS}}_{{.Arch}}"
    - type: go
      url: github.com/example/my-tool
```

## Library API

### Ensure and Run

```go
// Install a tool if needed, then run it
result, err := tools.Run(ctx, "golangci-lint", []string{"run", "./..."})

// Install a single tool by name
err := tools.Ensure(ctx, "golangci-lint")

// Install all configured tools
err := tools.EnsureAll(ctx)
```

### Run Options

```go
result, err := tools.Run(ctx, "my-tool", args,
    tools.WithDir("/path/to/workdir"),
    tools.WithEnv("FOO=bar"),
    tools.WithStdout(),          // Capture stdout in result.Stdout
    tools.WithStderr(),          // Capture stderr in result.Stderr
    tools.WithCombinedOutput(),  // Capture interleaved output in result.Combined()
)
```

### Run Arbitrary Binaries

```go
// Run a binary not in the tools config (e.g. "go" itself)
result, err := tools.RunBinary(ctx, "go", []string{"mod", "verify"})
```

### CI Cache Keys

```go
key, err := tools.CacheKey(".tools.yaml")
// Returns a SHA-256 hash of config content + GOOS + GOARCH
```

### Version Updates

```go
// Update all tools to latest versions
err := tools.UpdateAll(".tools.yaml")

// Update a single tool
err := tools.UpdateByName("golangci-lint", ".tools.yaml")
```

## Binary Layout

Installed binaries are placed under a platform-specific directory:

```
<toolsDir>/
  <GOOS>/
    <GOARCH>/
      golangci-lint-v2.9.0    # Versioned binary
      golangci-lint -> golangci-lint-v2.9.0  # Symlink
```

The tools directory is automatically prepended to `PATH` when `Ensure`, `EnsureAll`, or `Run` is called.

## License

Same license as parent project.
