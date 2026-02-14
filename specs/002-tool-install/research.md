# Research: Tool Installation Feature

**Feature**: 002-tool-install
**Date**: 2026-02-14

## R-001: Kubernetes API Machinery Dependencies

**Decision**: Use `k8s.io/apimachinery` as the sole K8s dependency for TypeMeta, scheme/codec, YAML serialization, defaulting, and validation. No other K8s modules required.

**Rationale**: The clusterawsadm pattern demonstrates that standalone versioned config files (not CRDs, not in-cluster) only need apimachinery. Adding `k8s.io/api` or `k8s.io/client-go` would pull in unnecessary controller/cluster dependencies.

**Alternatives considered**:
- `k8s.io/api` + `k8s.io/client-go`: Rejected — only needed for in-cluster communication
- `k8s.io/apiextensions-apiserver`: Rejected — only needed for CRD management
- Plain YAML parsing without K8s machinery: Rejected — loses typed schema validation, defaulting, and version conversion

**Key imports**:
- `k8s.io/apimachinery/pkg/apis/meta/v1` — TypeMeta
- `k8s.io/apimachinery/pkg/runtime` — runtime.Object, Scheme, SchemeBuilder
- `k8s.io/apimachinery/pkg/runtime/serializer` — CodecFactory
- `k8s.io/apimachinery/pkg/runtime/serializer/yaml` — GVK detection
- `k8s.io/apimachinery/pkg/util/wait` — ExponentialBackoff for retries

**Version**: `k8s.io/apimachinery v0.35.x` (latest stable, Go 1.25 compatible)

## R-002: Code Generation Tools

**Decision**: Use `controller-gen` for DeepCopy generation and `defaulter-gen` for defaults generation.

**Rationale**: These are the standard tools used by all K8s API Machinery projects (CAPA, Cluster API, etc.). controller-gen generates `zz_generated.deepcopy.go` from `+kubebuilder:object:generate=true` markers. defaulter-gen generates `zz_generated.defaults.go` from `+k8s:defaulter-gen=TypeMeta` markers.

**Alternatives considered**:
- Manual DeepCopy implementation: Rejected — error-prone and unmaintainable
- Code generation via `go generate`: Uses controller-gen and defaulter-gen under the hood anyway

**Versions**:
- `sigs.k8s.io/controller-tools/cmd/controller-gen@v0.19.0`
- `k8s.io/code-generator/cmd/defaulter-gen@v0.34.0`

## R-003: Network Retry with Exponential Backoff

**Decision**: Use `k8s.io/apimachinery/pkg/util/wait.ExponentialBackoff` for retry logic.

**Rationale**: Since we already depend on `k8s.io/apimachinery`, using its battle-tested `wait.ExponentialBackoff` avoids adding another dependency. It supports configurable initial duration, factor, jitter, max steps, and cap — all of which map to the spec's "exponential backoff with up to 3 retries" requirement.

**Alternatives considered**:
- `cenkalti/backoff/v4`: Rejected — adds unnecessary dependency when apimachinery provides the same functionality
- Custom implementation: Rejected — reinventing well-tested code
- No retry (rely solely on multi-source fallback): Rejected — user chose exponential backoff in clarification

**Configuration**:
```go
backoff := wait.Backoff{
    Duration: 1 * time.Second,
    Factor:   2.0,
    Jitter:   0.1,
    Steps:    3,
    Cap:      10 * time.Second,
}
```

## R-004: Archive Extraction

**Decision**: Use Go stdlib (`archive/tar`, `compress/gzip`, `archive/zip`) for archive extraction.

**Rationale**: The project needs to extract a single named binary from tar.gz or zip archives. Stdlib provides straightforward APIs for this specific use case with no additional dependencies. `mholt/archives` (already in go.mod as indirect) is a more comprehensive library but adds unnecessary complexity for our needs — we don't need format auto-detection, multiple compression formats, or streaming archive creation.

**Alternatives considered**:
- `mholt/archives`: Already indirect dep, but overkill. Supports many formats we don't need. Would move from indirect to direct dependency unnecessarily.
- Third-party tar/zip libraries: No clear advantage over stdlib for our use case.

**Implementation**: ~50 lines per format (tar.gz, zip). A small internal helper with format detection by file extension.

## R-005: HTTP Downloads and Checksum Verification

**Decision**: Use stdlib `net/http` for downloads and `crypto/sha256` with `io.TeeReader` for checksum verification.

**Rationale**: Binary downloads are straightforward HTTP GET requests. No progress bars needed (this is a build tool, not a user-facing download manager). The `io.TeeReader` pattern allows computing checksums during download without a second pass over the file.

**Alternatives considered**:
- `hashicorp/go-retryablehttp`: Adds dependency; retry is handled at a higher level (R-003)
- `grab` or similar download libraries: Overkill for binary downloads
- Post-download checksum verification: Less efficient — TeeReader computes during download

## R-006: URL Template Rendering

**Decision**: Use stdlib `text/template` for URL, binaryPath, and checksum URL template rendering.

**Rationale**: The template tags (`{{.Version}}`, `{{.VersionNum}}`, `{{.Major}}`, `{{.Minor}}`, `{{.Patch}}`, `{{.OS}}`, `{{.Arch}}`) are simple field substitutions that `text/template` handles natively. Version is parsed via semver for component access; OS/arch mapping is done before template execution by looking up `runtime.GOOS`/`runtime.GOARCH` in the configured maps.

**Alternatives considered**:
- `html/template`: Rejected — applies HTML escaping which would corrupt URLs
- String replacement (`strings.ReplaceAll`): Less flexible, doesn't support future template functions
- Custom template syntax: Rejected — Go templates are standard and well-understood

## R-007: kube-api-linter Integration

**Decision**: Use kube-api-linter as a golangci-lint v2 module plugin via `.custom-gcl.yml`.

**Rationale**: The module plugin approach is officially recommended by golangci-lint and avoids CGO/platform issues of Go plugins. It requires building a custom golangci-lint binary that includes the KAL linter.

**Alternatives considered**:
- Go plugin (`-buildmode=plugin`): Rejected — requires CGO, platform-specific, fragile version matching
- Standalone kube-api-linter binary: Could work for CI but doesn't integrate with existing golangci-lint workflow
- Skip API linting: Rejected — user explicitly requested it

**Configuration**:
```yaml
# .custom-gcl.yml
version: v2.5.0
name: golangci-lint-kube-api-linter
destination: ./bin
plugins:
  - module: 'sigs.k8s.io/kube-api-linter'
    version: 'v0.0.0-20260206102632-39e3d06a2850'
```

**Recommended linter subset** (for standalone config types without ObjectMeta):
- `jsontags`, `optionalorrequired`, `optionalfields`, `requiredfields`
- `commentstart`, `arrayofstruct`, `nobools`, `nofloats`, `integers`
- Disabled: `statusoptional`, `statussubresource` (controller-specific)

**Limitation**: No official releases — uses pseudo-versions. Pin version in CI.

## R-008: Semver Comparison for Version Updates

**Decision**: Use `hashicorp/go-version` for semver parsing and comparison across all ecosystems.

**Rationale**: Already an indirect dependency in go.mod (`v1.8.0`). Handles semver ranges, pre-releases, and tag prefix stripping. Needed by the update command to determine "latest version" across Go, npm, crates.io, PyPI, and RubyGems registries.

**Alternatives considered**:
- `Masterminds/semver/v3`: Also in go.mod indirect, but `hashicorp/go-version` has a simpler API for comparison
- `blang/semver/v4`: Another option, but already have hashicorp indirect
- stdlib string comparison: Insufficient for proper semver ordering

## Summary

| Decision | Choice | New Direct Dependency? |
|----------|--------|----------------------|
| K8s API Machinery | `k8s.io/apimachinery` | Yes |
| Code generation | controller-gen + defaulter-gen | Tools only (not in go.mod) |
| Network retry | `k8s.io/apimachinery/pkg/util/wait` | No (same module) |
| Archive extraction | stdlib `archive/tar`, `compress/gzip`, `archive/zip` | No |
| HTTP downloads | stdlib `net/http` | No |
| Checksum verification | stdlib `crypto/sha256` + `io.TeeReader` | No |
| URL templating | stdlib `text/template` | No |
| kube-api-linter | Module plugin via `.custom-gcl.yml` | Build tool only |
| Semver comparison | `hashicorp/go-version` | Promote from indirect |
