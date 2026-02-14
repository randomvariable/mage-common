# Data Model: Tool Installation

**Feature**: 002-tool-install
**Date**: 2026-02-14

## Entities

### ToolConfiguration (top-level API object)

The root configuration object loaded from `.tools.yaml`. Implements `runtime.Object` via embedded `TypeMeta`.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| apiVersion | string | Yes | — | `mage-common.randomvariable.co.uk/v1alpha1` |
| kind | string | Yes | — | `ToolConfiguration` |
| spec | ToolConfigurationSpec | Yes | — | Tool definitions and global settings |

**Validation**: apiVersion must match registered scheme; kind must be `ToolConfiguration`.

### ToolConfigurationSpec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| toolsDir | string | No | `hack/bin` | Base directory for installed binaries |
| tools | []Tool | Yes | — | List of tool definitions |

**Validation**: At least one tool must be defined. toolsDir must be a relative path.

### Tool

A single tool entry. Name is the unique identity key.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| name | string | Yes | — | Binary name (unique across config) |
| version | string | No | — | Pinned version (semver tag). Omit for local-only tools. |
| tagPrefix | string | No | `""` | Git tag prefix for version discovery (e.g., `kustomize/`) |
| sources | []ToolSource | Yes | — | Priority-ordered installation sources |

**Validation**: Name must be unique across all tools. At least one source required. Version required unless all sources are local path type.

**Identity**: `name` field. Used for install-by-name, run-by-name, update-by-name, cache key.

### ToolSource

A single installation method for a tool. Multiple sources per tool enable fallback.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| type | SourceType | Yes | — | Install type: `go`, `gem`, `npx`, `cargo`, `uvx`, `download` |
| url | string | Conditional | — | Go module URL or download URL template (required for go/download) |
| package | string | Conditional | — | Package name (required for gem/npx/cargo/uvx) |
| path | string | Conditional | — | Local path for in-tree tools, relative to Go module root (required for local builds) |
| binaryPath | string | No | tool name at archive root | Path to binary within archive (download type, supports template tags) |
| checksum | *Checksum | No | — | Integrity verification (download type only) |
| osMap | map[string]string | No | — | Map GOOS → URL OS string (download type) |
| archMap | map[string]string | No | — | Map GOARCH → URL arch string (download type) |
| overrides | []PlatformOverride | No | — | Per OS-arch field overrides (download type) |
| allowInsecure | bool | No | `false` | Permit HTTP URLs for this source (download type only). When false, non-HTTPS URLs are rejected at validation. |
| skipChecksum | bool | No | `false` | Skip checksum requirement for this download source. When false (default), a checksum field is required for download-type sources. Set to true for sources that don't publish checksums. |

**Validation**: Exactly one of `url`, `package`, or `path` must be set (mutual exclusion based on type). Template tags in `url` and `binaryPath` must be valid Go templates. For download-type sources, URLs must use HTTPS unless `allowInsecure` is true. For download-type sources, a checksum is required unless `skipChecksum` is true.

### PlatformOverride

Allows overriding download source fields for specific OS-architecture combinations.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| os | string | Yes | — | GOOS value to match |
| arch | string | Yes | — | GOARCH value to match |
| url | string | No | — | Override URL template |
| binaryPath | string | No | — | Override binary path in archive |
| checksum | *Checksum | No | — | Override checksum |

**Validation**: At least one of url, binaryPath, or checksum must be set.

### Checksum

Integrity verification for downloaded files.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| url | string | No | — | URL to checksums file (template tags supported) |
| inline | string | No | — | Inline hash: `<algorithm>:<hex>` (e.g., `sha256:<hex>`, `sha384:<hex>`, `sha512:<hex>`) |

**Validation**: Exactly one of url or inline must be set. For inline checksums, the algorithm prefix must be one of `sha256`, `sha384`, or `sha512`. Weaker algorithms (`md5`, `sha1`) are rejected at validation.

### SourceType (enum)

```
go | gem | npx | cargo | uvx | download
```

## Template Context (used in URL, binaryPath, checksum URL rendering)

### TemplateData

The data struct passed to Go `text/template` when rendering URL, binaryPath, and checksum URL templates. Built from the tool's version string + resolved platform values.

| Field | Type | Source | Example | Description |
|-------|------|--------|---------|-------------|
| Version | string | Tool.version as-is | `v2.4.0` | Full version string from config (preserves `v` prefix if present) |
| VersionNum | string | Version with `v` prefix stripped | `2.4.0` | Version without leading `v` — use when the URL path needs the bare number |
| Major | string | Semver parse | `2` | Major version component (empty string if version is not semver-parseable) |
| Minor | string | Semver parse | `4` | Minor version component (empty string if not parseable) |
| Patch | string | Semver parse | `0` | Patch version component (empty string if not parseable) |
| OS | string | runtime.GOOS resolved through osMap | `linux` | Target OS (after osMap lookup; raw GOOS if no map entry) |
| Arch | string | runtime.GOARCH resolved through archMap | `amd64` | Target architecture (after archMap lookup; raw GOARCH if no map entry) |

**Version parsing**: If the version string is parseable as semver (with or without `v` prefix), Major/Minor/Patch are populated. For non-semver versions (e.g., `latest`, date-based), these fields are empty strings and templates using them will render empty.

**Examples**:
- `version: v2.4.0` → Version=`v2.4.0`, VersionNum=`2.4.0`, Major=`2`, Minor=`4`, Patch=`0`
- `version: 1.32.0` → Version=`1.32.0`, VersionNum=`1.32.0`, Major=`1`, Minor=`32`, Patch=`0`
- `version: latest` → Version=`latest`, VersionNum=`latest`, Major=``, Minor=``, Patch=``

## Runtime Entities (not persisted)

### RunResult

Returned by the tool runner after executing a tool.

| Field | Type | Description |
|-------|------|-------------|
| ExitCode | int | Process exit code (0 = success) |
| Stdout | []byte | Captured stdout (if WithStdout() option used) |
| Stderr | []byte | Captured stderr (if WithStderr() option used) |
| Duration | time.Duration | Wall-clock execution time |

**Methods**: `Combined() []byte` — returns interleaved stdout+stderr (if WithCombinedOutput() used).

### RunOption (functional option)

| Option | Description |
|--------|-------------|
| WithStdout() | Capture stdout into RunResult |
| WithStderr() | Capture stderr into RunResult |
| WithCombinedOutput() | Capture interleaved stdout+stderr |
| WithEnv(...string) | Add environment variables |
| WithDir(string) | Set working directory |
| WithSecrets(...string) | Register values to redact from printed command line |
| WithoutFailOnNonZero() | Suppress error on non-zero exit |
| WithLogger(io.Writer) | Override output destination |

## State Transitions

### Tool Installation State

```
NOT_INSTALLED → INSTALLING → INSTALLED
                     ↓
                  FAILED → RETRY (up to 3x per source)
                              ↓
                          NEXT_SOURCE (fallback)
                              ↓
                          ALL_FAILED (reported in summary)
```

### Tool Version State

```
NOT_INSTALLED → install → CURRENT_VERSION
CURRENT_VERSION → config change → OUTDATED → reinstall → CURRENT_VERSION
CURRENT_VERSION → no config change → SKIP (idempotent)
```

## Relationships

```
ToolConfiguration 1──* Tool
Tool 1──* ToolSource
ToolSource 0──* PlatformOverride
ToolSource 0──1 Checksum
```

## Platform Directory Layout

```
<toolsDir>/
└── <GOOS>/
    └── <GOARCH>/
        ├── golangci-lint              → golangci-lint-v2.4.0 (symlink)
        ├── golangci-lint-v2.4.0       (versioned binary)
        ├── controller-gen             → controller-gen-v0.18.0 (symlink)
        ├── controller-gen-v0.18.0     (versioned binary)
        ├── rubocop                    (gem binstub)
        └── ruff                       (uvx shim)
```
