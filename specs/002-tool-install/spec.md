# Feature Specification: YAML-Configured Tool Dependency Installation

**Feature Branch**: `002-tool-install`
**Created**: 2026-02-14
**Status**: Draft
**Input**: User description: "YAML-configured development tool dependency installation with Go tool support and GitHub Actions caching"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Install All Tools from Config (Priority: P1)

A developer clones a project that uses this library and runs a single command to install all development tool dependencies defined in the project's `.tools.yaml` configuration file. Each tool is installed at the pinned version, placed in a platform-specific directory, and made available on PATH.

**Why this priority**: This is the core value proposition — developers get a reproducible, declarative tool environment with one command instead of manually installing each tool.

**Independent Test**: Can be fully tested by creating a `.tools.yaml` config with tools across multiple install types (e.g., a Go tool, a gem, and an npx package), running the install command, and verifying each binary exists at the expected path with the correct version.

**Acceptance Scenarios**:

1. **Given** a project with a `.tools.yaml` config listing tools across multiple install types (go, gem, npx, cargo, uvx) with pinned versions, **When** the developer runs the "install all tools" command, **Then** all tools are installed as versioned binaries in the platform-specific tools directory with symlinks pointing to the correct versions.
2. **Given** a project with tools already installed at the correct versions, **When** the developer runs the "install all tools" command again, **Then** no reinstallation occurs and the command completes quickly (idempotent).
3. **Given** a project with one tool installed at an older version, **When** the developer runs the "install all tools" command, **Then** only the outdated tool is reinstalled at the new version, existing up-to-date tools are untouched.

---

### User Story 2 - Install a Single Tool by Name (Priority: P1)

A developer needs only one specific tool (e.g., `golangci-lint`) and installs it by name rather than installing the entire toolset, saving time.

**Why this priority**: Essential for fast iteration — developers frequently need just one tool during a specific workflow step.

**Independent Test**: Can be tested by running the single-tool install command with a tool name from the config and verifying only that tool is installed.

**Acceptance Scenarios**:

1. **Given** a `.tools.yaml` config listing multiple tools, **When** the developer runs the install command for a single tool by name, **Then** only that tool is installed.
2. **Given** a tool name that does not exist in the `.tools.yaml` config, **When** the developer runs the install command for it, **Then** a clear error message is returned indicating the tool is not configured.

---

### User Story 3 - GitHub Actions Caching (Priority: P2)

A CI pipeline operator configures GitHub Actions to cache the installed tools directory so that subsequent workflow runs skip tool installation entirely when the tool configuration hasn't changed.

**Why this priority**: CI builds are the most frequent consumer of tool installation, and caching dramatically reduces build times and network usage.

**Independent Test**: Can be tested by running a GitHub Actions workflow twice with the same config and verifying the second run restores from cache and skips installation.

**Acceptance Scenarios**:

1. **Given** a GitHub Actions workflow using the provided cache configuration, **When** the workflow runs for the first time, **Then** tools are installed and the tools directory is cached with a key derived from the tool configuration content and platform.
2. **Given** a cached tools directory from a previous run with the same config, **When** the workflow runs again, **Then** the cache is restored and no tools are reinstalled.
3. **Given** a cached tools directory from a previous run, **When** the `.tools.yaml` config changes (tool added, version bumped), **Then** the cache key changes, a fresh install occurs, and the new state is cached.

---

### User Story 4 - Local Project Tool Builds (Priority: P2)

A project maintainer defines tools that are built from local source within the project (e.g., custom code generators), alongside remote tools, in the same `.tools.yaml` config.

**Why this priority**: Many projects have custom tooling that lives in-tree. Supporting local builds in the same config avoids a separate build system for project-specific tools.

**Independent Test**: Can be tested by adding a local tool entry pointing to a Go main package in the project and verifying it builds and is placed in the tools directory.

**Acceptance Scenarios**:

1. **Given** a `.tools.yaml` config entry with a local path instead of a remote URL, **When** the developer runs the install command, **Then** the tool is built from the local source and placed in the tools directory.
2. **Given** a local tool whose source has changed since last build, **When** the developer runs the install command, **Then** the tool is rebuilt with the latest source.

---

### User Story 5 - Multi-Source Fallback for Tools (Priority: P2)

A developer or CI engineer configures a tool with multiple installation sources in priority order. At home, the developer downloads a pre-built binary directly from a GitHub release for speed. In an enterprise CI environment where direct GitHub downloads are blocked, the system automatically falls back to `go install` via a configured GOPROXY.

**Why this priority**: Enterprise environments frequently restrict direct internet access, making a single installation method insufficient. A fallback chain ensures the same config works across open and restricted networks.

**Independent Test**: Can be tested by configuring a tool with two sources (download + go install), verifying the first source is used when available, then making the first source URL unreachable and verifying the fallback source succeeds.

**Acceptance Scenarios**:

1. **Given** a tool configured with a download source (priority 1) and a go install source (priority 2), **When** the download URL is reachable, **Then** the tool is downloaded directly and placed in the tools directory.
2. **Given** a tool configured with a download source (priority 1) and a go install source (priority 2), **When** the download URL is unreachable, **Then** the system falls back to `go install` and the tool is installed via the Go module proxy.
3. **Given** a tool configured with a download source using template tags for OS, architecture, and version, **When** the install runs on darwin/arm64, **Then** the URL is resolved with the correct platform values and the binary is downloaded.
4. **Given** a tool with only one source configured, **When** that source fails, **Then** the error is reported as normal (no fallback available).

---

### User Story 6 - Update Tools to Latest Versions (Priority: P2)

A developer or project maintainer runs an "update tools" command that queries each tool's upstream registry for the latest available version and updates the `.tools.yaml` config file with the new versions. This works across all install types — checking GitHub releases, Go module tags, npm registry, crates.io, PyPI, and RubyGems.

**Why this priority**: Keeping tools up-to-date is a recurring maintenance task. Automating version discovery across all ecosystems saves significant manual effort and reduces the risk of running outdated tools with known vulnerabilities.

**Independent Test**: Can be tested by configuring a tool at an older version, running the update command, and verifying the `.tools.yaml` config is updated to a newer version while the on-disk tool remains at the old version until reinstalled.

**Acceptance Scenarios**:

1. **Given** a `.tools.yaml` config with a Go tool pinned to an older version, **When** the developer runs the update command, **Then** the config file is updated with the latest available version from the Go module proxy or GitHub releases.
2. **Given** a `.tools.yaml` config with tools across multiple install types, **When** the developer runs the update command, **Then** each tool's version is checked against its respective upstream registry (npm for npx, crates.io for cargo, PyPI for uvx, RubyGems for gem, GitHub/Go proxy for go).
3. **Given** a tool with a tag prefix (e.g., `kustomize/v5.7.1`), **When** the update command checks for new versions, **Then** it correctly filters tags by prefix and identifies the latest version.
4. **Given** a tool pinned to `latest`, **When** the update command runs, **Then** that tool is skipped (already tracking latest).
5. **Given** the update command completes, **When** the developer reviews the changes, **Then** only the version fields in the `.tools.yaml` config are modified — no tools are reinstalled until the install command is run separately.

---

### User Story 8 - Run Installed Tools with Observable Output (Priority: P1)

A Mage target author uses a helper to execute an installed tool, getting the full command line printed to the console (with secrets redacted), live-streamed stdout/stderr output, and programmatic access to the captured output and exit code for conditional logic.

**Why this priority**: Every Mage target that uses an installed tool needs to run it. A consistent runner helper avoids duplicated exec boilerplate across targets, ensures commands are always logged for debugging, and prevents accidental secret leakage in CI logs.

**Independent Test**: Can be tested by running a tool via the helper and verifying: (1) the command line is printed, (2) stdout and stderr are streamed in real time, (3) the caller can access captured stdout, stderr, or combined output, and (4) the exit code is available for inspection.

**Acceptance Scenarios**:

1. **Given** an installed tool in the tools directory, **When** a Mage target runs it via the tool runner helper, **Then** the full command line (binary path + arguments) is printed to the console before execution begins.
2. **Given** a command line that contains values matching configured secrets, **When** the command line is printed, **Then** secret values are redacted (optionally reusing the secrets redaction from the config package).
3. **Given** a running tool that produces output on stdout and stderr, **When** the tool is executing, **Then** both streams are forwarded to the console (or logger) in real time — the caller does not need to wait for the process to finish to see output.
4. **Given** a Mage target that needs to parse tool output, **When** the tool finishes, **Then** the caller can access the captured stdout, stderr, or combined output as byte slices or strings.
5. **Given** a tool that exits with a non-zero status, **When** the caller checks the result, **Then** the exit code is available and the caller can decide whether to fail the Mage target or handle the error.

---

### User Story 7 - Documentation and Examples (Priority: P3)

A developer or CI engineer new to the project reads documentation that explains how to configure tools, how caching works, and sees ready-to-use examples for GitHub Actions workflows.

**Why this priority**: Good documentation reduces onboarding friction and support burden, but the system must work before docs matter.

**Independent Test**: Can be tested by having a new team member follow the documentation to set up tool installation in a fresh project.

**Acceptance Scenarios**:

1. **Given** the published documentation, **When** a developer follows the setup guide for a new project, **Then** they can configure a `.tools.yaml` file and install tools within 5 minutes.
2. **Given** the GitHub Actions examples in the documentation, **When** a CI engineer copies and adapts the example workflow, **Then** caching works correctly on the first attempt.

---

### Edge Cases

- What happens when a tool's Go module URL is unreachable (network failure)?
- What happens when a version tag does not exist for a configured tool?
- What happens when two tools have the same binary name but different URLs? (Rejected at config validation — tool names must be unique.)
- What happens on unsupported platforms (e.g., Windows ARM)?
- What happens when the `.tools.yaml` config file is missing or malformed?
- What happens when disk space is insufficient during installation?
- What happens when Go is not installed on the system?
- What happens when a required runtime (Ruby, Node.js, Rust/Cargo, Python/uv) is not installed for a tool that needs it?
- What happens when `npx` or `uvx` is used with a tool that has transitive dependency conflicts?
- What happens when a Cargo crate requires a C toolchain for native extensions?
- What happens when all sources in a tool's priority list fail?
- What happens when a downloaded binary doesn't match the expected platform or is corrupt?
- What happens when the download source returns a tarball/zip vs a raw binary?
- What happens when a download checksum does not match (supply chain integrity)?
- What happens when the checksums URL is unreachable but the binary URL is reachable?
- What happens when the update command cannot reach an upstream registry for one tool?
- What happens when the update command finds no newer version available?
- What happens when the `.tools.yaml` file has an unrecognised apiVersion or kind?
- What happens when a required field is missing from a tool definition in the `.tools.yaml` file?
- What happens when the current platform has no entry in the OS or architecture map and no override?
- What happens when a platform override specifies only some fields (e.g., URL but not binaryPath)?
- What happens when the tool runner is asked to run a tool that is not installed?
- What happens when a tool produces very large output (memory pressure from capture buffers)?
- What happens when a secret value appears as a substring of a non-secret argument?

## Clarifications

### Session 2026-02-14

- Q: When one tool fails during a batch install, should the system stop or continue? → A: Configurable — default continue-on-failure with collected error summary, with a strict/fail-fast flag to stop on first failure.
- Q: Should the versioned-symlink pattern apply uniformly to all install types, or should each type use its native mechanism? → A: Native mechanism per type — each ecosystem uses its own convention for idempotency (symlinks for go/cargo, binstubs for gem, shims for npx/uvx), but all must place an executable with the tool's name in the unified tools directory. Version check adapts per install type.
- Q: When a tool's required runtime is missing, should the system pre-check or let the command fail? → A: Pre-check — verify the runtime exists on PATH before attempting installation and produce a clear, actionable error message (e.g., "cargo-type tool 'ripgrep' requires Rust/Cargo but `cargo` was not found on PATH").
- Q: Should tools be installed in parallel or sequentially? → A: Parallel across types, sequential within type — different ecosystems run concurrently (Go and Cargo simultaneously) but tools within the same install type run one at a time to avoid package manager contention.
- Q: How should transient network failures be handled within a single source attempt? → A: Exponential backoff with up to 3 retries per source before marking that source as failed and moving to the next fallback (if any).
- Q: How is the target binary located within a downloaded archive? → A: Optional `binaryPath` field with template tag support (version, OS, arch). Defaults to the tool name at the archive root when omitted.
- Q: Must tool names be unique in the configuration, or can duplicates exist? → A: Unique names enforced — config validation rejects duplicate tool names at load time. The tool name is the identity key for install, run, update, and cache operations.
- Q: Should download URLs be restricted to HTTPS only? → A: HTTPS by default — download URLs must use HTTPS. HTTP URLs are rejected at config validation unless the source explicitly opts in with an `allowInsecure: true` field. Checksums URLs follow the same rule.
- Q: Should checksum verification be mandatory or optional for download-type sources? → A: Mandatory by default — download-type sources must include a checksum (URL or inline). Config validation rejects download sources without a checksum unless the source explicitly opts out with a `skipChecksum: true` field for sources that don't publish checksums.
- Q: Should the spec include explicit filesystem safety requirements for archive extraction? → A: Yes, full coverage — the archive extractor must reject entries with `../` or absolute paths (Zip Slip), reject symlink entries pointing outside the extraction directory, and validate that `binaryPath` resolves within the archive root. Covers path traversal, symlink attacks, and binaryPath traversal.
- Q: Should the spec define HTTP redirect security requirements for downloads? → A: Yes — the downloader must reject HTTPS→HTTP redirects (downgrade attack prevention) while allowing HTTPS→HTTPS redirects normally.
- Q: Should the spec require SHA-256 as the minimum checksum algorithm and explicitly disallow weaker algorithms? → A: SHA-256 minimum — accept SHA-256 and stronger (SHA-384, SHA-512) but reject MD5 and SHA-1 at config validation with a clear error.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST read tool definitions from a dedicated `.tools.yaml` configuration file with a versioned API schema (Kubernetes-style `apiVersion`/`kind`). The configuration uses Kubernetes API Machinery for loading, schema validation, defaulting, and future version conversion. Each tool has a name, version, and one or more sources listed in priority order. Each source specifies an install type and type-specific fields (URL/package name, local path, download URL template, etc.).
- **FR-035**: The configuration schema MUST use Kubernetes API Machinery conventions: `TypeMeta` (apiVersion, kind) for version identification, typed Go structs for schema enforcement, and a scheme/codec pattern for serialization and deserialization.
- **FR-036**: The configuration MUST support schema defaulting — omitted optional fields are populated with sensible defaults during loading (e.g., default tools directory path, default install type for simple entries).
- **FR-037**: The configuration MUST be validated after loading, rejecting invalid entries (missing required fields, unknown install types, conflicting fields, duplicate tool names) with clear error messages referencing the problematic tool entry. Tool names MUST be unique within the configuration — the name serves as the identity key for all operations (install, run, update, cache).
- **FR-002**: System MUST install Go tools using `go install <url>@<version>` for remote tools with pinned versions.
- **FR-003**: System MUST build Go tools from local project source using `go build` when a local path is specified instead of a URL.
- **FR-016**: System MUST install Ruby gem tools using Bundler binstubs, placing the resulting executables in the tools directory.
- **FR-017**: System MUST install Node.js tools using `npx` with a pinned package version, making the tool available as a binary in the tools directory.
- **FR-018**: System MUST install Rust tools using `cargo install` with a pinned version, placing the binary in the tools directory.
- **FR-019**: System MUST install Python tools using `uvx` (from the `uv` package manager) with a pinned version, making the tool available in the tools directory.
- **FR-004**: System MUST store installed tool binaries in a platform-specific directory structure: `<tools-dir>/<GOOS>/<GOARCH>/`.
- **FR-005**: System MUST use each install type's native versioning mechanism to track installed versions and enable idempotent installs. For Go and Cargo, this is versioned binary naming with symlinks (e.g., `tool` → `tool-v1.2.3`). For gem, npx, and uvx, the system uses each ecosystem's native approach (binstubs, shims) while ensuring an executable with the tool's name is present in the tools directory.
- **FR-006**: System MUST skip installation of a tool if it detects the correct version is already installed, using the install-type-appropriate version check.
- **FR-007**: System MUST support installing all configured tools in a single operation. Tools of different install types run concurrently (e.g., Go and Cargo in parallel), while tools within the same install type run sequentially to avoid package manager contention.
- **FR-008**: System MUST support installing a single tool by name.
- **FR-009**: System MUST return a clear error when a requested tool name is not found in the configuration.
- **FR-022**: System MUST verify that the required runtime for each tool's install type is available on PATH before attempting installation, and produce a clear, actionable error message identifying the missing runtime if it is not found.
- **FR-020**: System MUST by default continue installing remaining tools when one tool fails, collecting all errors and reporting a summary at the end.
- **FR-021**: System MUST support a strict/fail-fast mode that stops installation on the first tool failure.
- **FR-023**: System MUST support a `download` install type that fetches a pre-built binary from a URL. The URL MUST support Go template tags with the following context fields: `{{.Version}}` (full version as specified in config, e.g., `v2.4.0`), `{{.VersionNum}}` (version with any `v` prefix stripped, e.g., `2.4.0`), `{{.Major}}`, `{{.Minor}}`, `{{.Patch}}` (semver components when parseable), `{{.OS}}` (resolved OS name), and `{{.Arch}}` (resolved architecture name). The version field in the config stores the full version string including any `v` prefix.
- **FR-038**: For download-type sources, the system MUST support an OS name map that translates the system's OS identifier to the string used in the download URL (e.g., `darwin` → `macOS`, `linux` → `Linux`). When no map is provided, the system's native OS identifier is used.
- **FR-039**: For download-type sources, the system MUST support an architecture name map that translates the system's architecture identifier to the string used in the download URL (e.g., `amd64` → `x86_64`, `arm64` → `aarch64`). When no map is provided, the system's native architecture identifier is used.
- **FR-040**: For download-type sources, the system MUST support per OS-architecture overrides that allow replacing any source field (URL, binary path, checksum) for a specific platform combination. This handles cases where a project uses inconsistent URL patterns across platforms (e.g., a universal binary on macOS but arch-specific on Linux, or entirely different archive naming).
- **FR-050**: For download-type sources, the system MUST require HTTPS for all URLs (download URL, checksums URL). HTTP URLs MUST be rejected at config validation with a clear error. A per-source `allowInsecure` boolean field (default `false`) MAY be set to `true` to permit HTTP URLs for specific sources (e.g., internal mirrors without TLS).
- **FR-053**: The HTTP downloader MUST reject redirects from HTTPS to HTTP (transport downgrade attack). HTTPS→HTTPS redirects MUST be followed normally. When `allowInsecure` is true for a source, HTTP→HTTP redirects are also permitted.
- **FR-048**: For any network operation (download, registry query, module fetch), the system MUST retry transient failures using exponential backoff with up to 3 retries before marking the operation as failed. This applies per-source — after retries are exhausted, multi-source fallback (FR-024) proceeds to the next source if available.
- **FR-024**: System MUST support multiple sources per tool, defined as a priority-ordered list. The system tries the first source and falls back to subsequent sources if installation fails.
- **FR-025**: When a tool has multiple sources and the higher-priority source fails, the system MUST log the failure reason and attempt the next source in the list. If all sources fail, the tool is reported as failed.
- **FR-026**: For download-type sources, the system MUST handle both raw binaries and common archive formats (e.g., tar.gz, zip) and extract the tool binary from the archive.
- **FR-051**: The archive extractor MUST reject archive entries containing path traversal sequences (`../`) or absolute paths, preventing Zip Slip attacks. Archive entries that are symlinks pointing outside the extraction directory MUST also be rejected. These checks MUST apply to all supported archive formats (tar.gz, zip).
- **FR-052**: The `binaryPath` field (FR-049) MUST be validated to ensure it resolves within the archive root after template rendering. A `binaryPath` containing `../` or resolving to a path outside the archive MUST be rejected at extraction time with a clear error.
- **FR-049**: For download-type sources, the system MUST support an optional `binaryPath` field that specifies the path to the binary within an archive. The field MUST support the same template context as the URL (all version, OS, and architecture fields from FR-023). When omitted, the system defaults to looking for a binary matching the tool name at the archive root.
- **FR-027**: For download-type sources, the system MUST require a checksum field (URL or inline) by default. Config validation MUST reject download sources that omit the checksum unless the source explicitly sets `skipChecksum: true`. When a checksum is provided, the system MUST verify the downloaded file's integrity against the checksum before extracting or installing, and fail with a clear error if the checksum does not match.
- **FR-028**: The checksum field for download sources MUST support a URL to a checksums file (e.g., GitHub release SHA256SUMS) as well as inline hash values, with the hash algorithm specified as a prefix (e.g., `sha256:<hex>`). Checksum URLs MUST support the same Go template context as download URLs (all version, OS, and architecture fields from FR-023). The system MUST accept SHA-256, SHA-384, and SHA-512 algorithms. Weaker algorithms (MD5, SHA-1) MUST be rejected at config validation with a clear error identifying the unsupported algorithm.
- **FR-029**: System MUST provide an "update tools" command that queries each tool's upstream registry for the latest available version and updates the version field in the `.tools.yaml` config file without reinstalling tools.
- **FR-030**: The update command MUST support all install types by querying the appropriate upstream source: GitHub releases/tags for go and download types, npm registry for npx, crates.io for cargo, PyPI for uvx, and RubyGems for gem.
- **FR-031**: The update command MUST respect tag prefixes when determining the latest version for tools that use them.
- **FR-032**: The update command MUST skip tools pinned to `latest` or local-path-only tools that have no upstream version.
- **FR-033**: The update command MUST support updating a single tool by name, or all tools at once.
- **FR-034**: When updating a download-type source that has a checksums URL, the update command SHOULD also update the checksum to match the new version's checksums file.
- **FR-010**: System MUST support the `latest` keyword as a version. Per install type: for `go`, resolves to `go install <url>@latest`; for `gem`, installs the latest gem version; for `npx`, runs without a version pin; for `cargo`, installs the latest crate version; for `uvx`, installs the latest PyPI version. The `download` type MUST NOT support `latest` (download URLs require an explicit version for template rendering). Config validation MUST reject `latest` for download-type sources.
- **FR-011**: System MUST support a tag prefix field for tools whose Git tags use a path prefix (e.g., `kustomize/v5.7.1`).
- **FR-012**: System MUST make installed tools available on PATH by providing the tools directory path for callers to prepend.
- **FR-013**: System MUST provide a deterministic cache key derivation mechanism based on the `.tools.yaml` configuration content and the target platform (OS/architecture), suitable for use in CI caching systems.
- **FR-014**: System MUST include documentation with examples for GitHub Actions workflow caching, including cache key configuration.
- **FR-015**: System MUST provide a reusable GitHub Actions composite action or documented pattern for tool caching that teams can adopt directly.
- **FR-041**: System MUST provide a tool runner helper that executes an installed tool by name, resolving its binary path from the tools directory.
- **FR-042**: The tool runner MUST print the full command line (binary path and all arguments) before execution, for debugging and auditability.
- **FR-043**: The tool runner MUST support secrets redaction in command-line logging — any argument value matching a configured secret is replaced with a placeholder (e.g., `***`). This MAY reuse the secrets redaction facility from the config package (001-viper-config) if available.
- **FR-044**: The tool runner MUST stream stdout and stderr to the console (or a configured logger) in real time during execution, so output is visible immediately without waiting for the process to complete.
- **FR-045**: The tool runner MUST allow the caller to capture stdout, stderr, or combined output as byte slices or strings, using multi-writers so that streaming and capture happen simultaneously.
- **FR-046**: The tool runner MUST by default return an error when the process exits with a non-zero status. A functional option MUST be provided to suppress this behaviour, returning a normal result instead so the caller can inspect the exit code and decide how to handle it.
- **FR-047**: The tool runner MUST capture execution wall-clock duration for each tool invocation and expose it in the result, so it is available for future metrics and tracing instrumentation (e.g., OpenTelemetry).
- **FR-054**: The checksums file parser MUST support GNU coreutils format (`<hash>  <filename>` or `<hash> <filename>`). The correct entry MUST be matched by comparing the filename component of the rendered download URL to the filename in each checksums line. If no matching line is found, or if multiple ambiguous matches exist, the system MUST return a clear error.
- **FR-055**: Extracted binaries MUST be set to mode 0755 (owner read/write/execute, group and others read/execute). World-writable permissions MUST NOT be used. The tools directory MUST be created with mode 0750 (owner read/write/execute, group read/execute).
- **FR-056**: Downloads MUST use atomic file operations — write to a temporary file in the same directory, then rename to the target path on success. Partial downloads MUST be cleaned up on failure (temporary file removed).
- **FR-057**: The retry mechanism (FR-048) MUST only retry on transient failures: HTTP 5xx status codes, network connection errors, and timeouts. HTTP 4xx status codes (including 401 Unauthorized, 403 Forbidden, 404 Not Found) MUST NOT be retried.
- **FR-058**: All network operations MUST respect the caller's `context.Context` for cancellation and timeout. The HTTP client MUST use Go's default TLS configuration, which uses system CA roots. Custom CA support is available via standard Go environment variables (`SSL_CERT_FILE`, `SSL_CERT_DIR`).
- **FR-059**: The tool runner MUST be safe for concurrent use — each `Run()` call is stateless and spawns an independent OS process. Callers MAY invoke `Run()` concurrently for different tools or for the same tool.
- **FR-060**: Secrets redaction (FR-043) applies to command-line logging only. Captured stdout/stderr in RunResult contains raw tool output — the runner does not redact tool output, as this would require understanding each tool's output format. Secrets redaction uses simple string replacement; if a secret value appears as a substring of a non-secret argument, it is still redacted (false positives are acceptable for security).

### Key Entities

- **ToolConfiguration**: The top-level versioned API object (has apiVersion, kind) containing the spec for all tool definitions and global settings (e.g., tools directory path).
- **Tool Definition**: A single tool entry within the configuration — name, version, and one or more sources in priority order.
- **Tool Source**: A single installation method for a tool — install type (go, gem, npx, cargo, uvx, download), plus type-specific fields (URL, package name, local path, download URL template, tag prefix, checksum, binaryPath).
- **Tools Directory**: Platform-specific directory holding installed binaries and version symlinks.
- **Cache Key**: A derived identifier representing the current state of tool configuration plus platform, used for CI cache invalidation.
- **Tool Runner**: A helper that executes installed tools with command-line logging (with secrets redaction), real-time output streaming, output capture, and exit code inspection.

## Assumptions

- Go is installed and available on PATH on the target system for Go-type tools.
- Ruby and Bundler are installed and available on PATH for gem-type tools.
- Node.js and npx are installed and available on PATH for npx-type tools.
- Rust and Cargo are installed and available on PATH for cargo-type tools.
- Python and uv (uvx) are installed and available on PATH for uvx-type tools.
- The tool configuration is a standalone `.tools.yaml` file with a versioned API schema, loaded via Kubernetes API Machinery (not Viper). This is separate from the project's general Viper-based `.mage.yaml` config (001-viper-config).
- Tools installed via `go install` use CGO_ENABLED=0 for static binaries by default.
- The tools directory is project-local (not global), under a conventional path like `hack/bin/` or configurable.
- The mage-common repository dogfoods its own tool installation system, consuming its own `.tools.yaml` in its Mage targets and CI pipelines.
- Each install type is responsible for placing its binary/shim into the shared tools directory; the system does not manage runtime installation itself.
- The versioned API types are linted with [kube-api-linter](https://github.com/kubernetes-sigs/kube-api-linter) to enforce Kubernetes API conventions.
- The `path` field for local builds is relative to the Go module root (where `go.mod` lives), consistent with standard `go build` path resolution.
- The tool runner inherits the full process environment by design — tools require access to environment variables (PATH, GOPATH, HOME, etc.) for correct operation. The caller controls environment content before invoking tool operations. `WithSecrets()` redacts values from logging; `WithEnv()` adds variables. No environment sanitisation is performed by the runner.
- Binary signing and provenance verification (SLSA, Sigstore/cosign) are out of scope for the initial v1alpha1 release. The system relies on checksum verification (FR-027/FR-028) for download integrity. Signing support may be added in future versions as an optional verification layer.
- The system delegates trust to each package manager ecosystem — `go install`, `cargo install`, `npx`, `gem`, and `uvx` commands execute with the same trust posture as manual developer usage. The security implications of each ecosystem (arbitrary code execution during installation via `init()` functions, `build.rs` scripts, npm lifecycle hooks, `extconf.rb`, etc.) are accepted risks inherent to using these tools. `CGO_ENABLED=0` for Go tools is primarily a portability requirement (static binaries) with a secondary benefit of avoiding dynamic library loading.
- This module follows Go module versioning: v0.x releases carry no API stability guarantees. The v1alpha1 API version follows Kubernetes versioning conventions where alpha = breaking changes expected. Schema evolution uses additive-only field changes within a version; breaking changes require a new API version (e.g., v1beta1). Deprecation policy will be defined at the v1beta1 milestone.
- The Updater logs progress per tool via the Logger interface (tool name, old version, new version, skipped/failed). The return type is `error` only — callers needing structured update results can diff the modified config file. This matches the build tool convention where logging is the primary feedback mechanism.
- Download URLs in `.tools.yaml` MUST NOT contain authentication secrets (API tokens, access keys). Private registries requiring authentication SHOULD use environment-variable-based mechanisms (e.g., `GITHUB_TOKEN` in HTTP headers, `.netrc`, Git credential helpers) rather than URL-embedded credentials.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can install all configured tools with a single command and have them available on PATH within 2 minutes on a standard internet connection.
- **SC-002**: Repeated tool installation with no configuration changes completes in under 5 seconds (cache hit / idempotent skip).
- **SC-003**: CI pipeline tool installation time is reduced by 80% or more on cache-hit runs compared to cold installs.
- **SC-004**: A new developer can configure tool installation for a fresh project within 5 minutes by following the documentation.
- **SC-005**: Cache key changes correctly whenever any tool name, version, URL, or platform changes, achieving 100% cache invalidation accuracy.
- **SC-006**: The system supports at least linux/amd64 and darwin/arm64 platforms without configuration changes.
- **SC-007**: The mage-common repository itself uses its own tool configuration and installation system (dogfooding), with its own `.tools.yaml` consumed by its Mage targets and validated in CI.
