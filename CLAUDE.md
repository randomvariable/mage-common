# mage-common Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-02-14

## Active Technologies
- Go 1.25.7 (from go.mod) + `sigs.k8s.io/kind` (cluster operations), `k8s.io/apimachinery` (TypeMeta), `k8s.io/client-go` (node readiness), `github.com/magefile/mage` (targets), `github.com/spf13/viper` + `pflag` (config integration) (003-kind-cluster-mage)
- `.kind-cluster.yaml` config file in project root (read-only) (003-kind-cluster-mage)

- Go 1.25.4 (002-tool-install)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.25.4

## Code Style

Go 1.25.4: Follow standard conventions

## Recent Changes
- 003-kind-cluster-mage: Added Go 1.25.7 (from go.mod) + `sigs.k8s.io/kind` (cluster operations), `k8s.io/apimachinery` (TypeMeta), `k8s.io/client-go` (node readiness), `github.com/magefile/mage` (targets), `github.com/spf13/viper` + `pflag` (config integration)

- 002-tool-install: Added Go 1.25.4

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
