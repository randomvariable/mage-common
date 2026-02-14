# Quickstart: Tool Installation

## 1. Create `.tools.yaml`

In your project root, create a `.tools.yaml` file:

```yaml
apiVersion: mage-common.randomvariable.co.uk/v1alpha1
kind: ToolConfiguration
spec:
  toolsDir: hack/bin
  tools:
    - name: golangci-lint
      version: v2.4.0
      sources:
        - type: download
          url: "https://github.com/golangci/golangci-lint/releases/download/{{.Version}}/golangci-lint-{{.VersionNum}}-{{.OS}}-{{.Arch}}.tar.gz"
          binaryPath: "golangci-lint-{{.VersionNum}}-{{.OS}}-{{.Arch}}/golangci-lint"
          checksum:
            url: "https://github.com/golangci/golangci-lint/releases/download/{{.Version}}/golangci-lint-{{.VersionNum}}-checksums.txt"
        - type: go
          url: github.com/golangci/golangci-lint/v2/cmd/golangci-lint
    - name: controller-gen
      version: v0.18.0
      sources:
        - type: go
          url: sigs.k8s.io/controller-tools/cmd/controller-gen
```

## 2. Add to your Magefile

```go
//go:build mage

package main

import (
    "github.com/randomvariable/mage-common/tools"
)

// Tools installs all development tools.
func Tools() error {
    return tools.InstallAll()
}

// Lint runs golangci-lint via the tool runner.
func Lint() error {
    result, err := tools.Run(context.Background(), "golangci-lint", []string{"run", "./..."})
    if err != nil {
        return err
    }
    return nil
}
```

## 3. Run

```bash
# Install all tools
mage tools

# Install a single tool
mage tools:install golangci-lint

# Update all tools to latest versions
mage tools:update
```

## 4. GitHub Actions Caching

```yaml
- name: Get tools cache key
  id: tools-key
  run: echo "key=$(mage tools:cachekey)" >> $GITHUB_OUTPUT

- uses: actions/cache@v4
  with:
    path: hack/bin/
    key: tools-${{ runner.os }}-${{ runner.arch }}-${{ steps.tools-key.outputs.key }}

- name: Install tools
  run: mage tools
```

## 5. Multi-Source Fallback

Tools try sources in order. If a fast download fails, it falls back to `go install`:

```yaml
- name: golangci-lint
  version: v2.4.0
  sources:
    - type: download        # Priority 1: fast binary download
      url: "https://github.com/..."
    - type: go              # Priority 2: fallback via Go module proxy
      url: github.com/golangci/golangci-lint/v2/cmd/golangci-lint
```

## 6. Platform-Specific Downloads

For tools with inconsistent naming across platforms:

```yaml
- name: kubectl
  version: v1.32.0
  sources:
    - type: download
      url: "https://dl.k8s.io/release/{{.Version}}/bin/{{.OS}}/{{.Arch}}/kubectl"
      osMap:
        darwin: darwin
        linux: linux
      archMap:
        amd64: amd64
        arm64: arm64
      overrides:
        - os: darwin
          arch: arm64
          url: "https://dl.k8s.io/release/{{.Version}}/bin/darwin/arm64/kubectl"
```
