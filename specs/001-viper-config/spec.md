# Feature Specification: Viper Configuration Package

**Feature Branch**: `001-viper-config`
**Created**: 2026-02-14
**Status**: Draft
**Input**: User description: "viper"

## Clarifications

### Session 2026-02-14

- Q: Should the config package validate that all command-line arguments match expected parameters, or allow any argument to pass through silently? → A: Viper default: silently ignore unknown arguments (follows Viper's standard behavior)
- Q: Should the config package log information about configuration loading? → A: Package logs via optional logger interface (caller supplies logger). Silent if no logger provided. Future: integrate with planned mage logging library
- Q: Should the package provide mechanisms to prevent accidental logging of sensitive configuration values? → A: Both manual and automatic - allow marking keys as sensitive + auto-detect common patterns (token, password, key, secret, credential). Redact as "[REDACTED]" in logs
- Q: What should the logger interface contract be? → A: Option B - Single method: Log(level Level, msg string, attrs ...any) - minimal interface, maximum adapter flexibility
- Q: How should sensitive configuration values be detected for redaction? → A: Use gitleaks library for proven secret detection - leverage battle-tested regex patterns and entropy analysis
- Q: When can sensitive keys be marked for redaction? → A: Option C - During Init() via functional options pattern (e.g., config.Init(config.WithSensitiveKeys("api.token")))
- Q: What events should be logged during configuration loading? → A: Option B - Standard logging (errors + config source confirmation on success) - log errors when they occur, log successful source (file/env vars) on completion, nothing else
- Q: What is the error handling contract when no logger is provided? → A: Option A - Init() returns error for any config-related failure, never writes to stdout/stderr when no logger provided (silent operation means no console output, errors still returned to caller)
## User Scenarios & Testing *(mandatory)*

### User Story 1 - Load Standard Configuration (Priority: P1)

A developer imports the config package to load their project's `config.yaml` file with standardized patterns, handling missing files gracefully and supporting environment variable overrides.

**Why this priority**: This is the foundational functionality that all other features depend on. Without standard config loading, developers cannot leverage Viper consistently across projects.

**Independent Test**: A Go program can call `config.Init()` and successfully read values from `config.yaml` or environment variables. The program continues to work even when config.yaml doesn't exist.

**Acceptance Scenarios**:

1. **Given** a project with `config.yaml` in the root, **When** developer calls `config.Init()`, **Then** configuration values are loaded and accessible via standard Viper methods
2. **Given** a project without `config.yaml`, **When** developer calls `config.Init()`, **Then** the function returns gracefully without error and config values default to environment variables or zero values
3. **Given** a config value set in both `config.yaml` and environment variable, **When** developer reads the value, **Then** environment variable takes precedence
4. **Given** nested configuration keys (e.g., `tools.golangci-lint.version`), **When** developer accesses them, **Then** values are retrieved correctly using Viper's dot notation

---

### User Story 2 - Integrate with Mage (Priority: P1)

A developer using Mage can integrate the config package to prevent flag parsing conflicts between Viper/Cobra and Mage, enabling clean magefile execution.

**Why this priority**: Without this, Mage targets break when Viper is used, making the library unusable in its primary context. This is a critical blocker for adoption.

**Independent Test**: A magefile with `config.CleanOSArgs()` in init() can successfully call Mage targets that use Viper for configuration without flag parsing errors.

**Acceptance Scenarios**:

1. **Given** a magefile with Viper-based configuration, **When** developer calls `mage targetName`, **Then** Mage executes without flag parsing conflicts
2. **Given** Mage's `-v` verbose flag is used, **When** developer runs `mage -v targetName`, **Then** the verbose flag is preserved and Mage runs with verbose output
3. **Given** a magefile calls `config.CleanOSArgs()` before Viper initialization, **When** Mage targets use Viper to read config, **Then** configuration loads correctly without interference

---

### User Story 3 - Named Arguments for Mage Targets (Priority: P2)

A developer can define Mage target functions that accept named arguments (e.g., `mage build --platform=linux --arch=amd64`) instead of positional arguments, improving usability and self-documentation.

**Why this priority**: This solves a major Mage limitation but isn't required for basic config loading. It significantly improves developer experience but can be added after core functionality works.

**Independent Test**: A Mage target with named parameters (using Viper binding) can be called with `--key=value` syntax, and values are correctly passed to the target function.

**Acceptance Scenarios**:

1. **Given** a Mage target function with parameters, **When** developer calls `mage build --image=webapp --tag=v1.0`, **Then** the function receives `image="webapp"` and `tag="v1.0"` from Viper
2. **Given** a named argument has a default value in config.yaml, **When** developer omits that argument, **Then** the default value from config is used
3. **Given** a named argument is provided both in config.yaml and command line, **When** developer calls the target, **Then** command line value overrides config file value

---

### User Story 4 - Cross-Project Consistency (Priority: P3)

A developer working across multiple projects experiences consistent configuration patterns, reducing cognitive load and setup time.

**Why this priority**: This is a quality-of-life improvement that becomes valuable after the core features are stable. It's about long-term maintainability rather than immediate functionality.

**Independent Test**: The same config package code works identically when imported by multiple different projects without modification.

**Note**: See `.private/project-references.md` for specific target projects (internal use only).

**Acceptance Scenarios**:

1. **Given** the config package is imported by three different projects, **When** each project calls `config.Init()`, **Then** all projects load their respective config.yaml files with identical behavior
2. **Given** a developer switches between projects, **When** they read the magefile init() pattern, **Then** they see the same familiar `config.Init()` and `config.CleanOSArgs()` calls

---

### Edge Cases

- What happens when config.yaml contains invalid YAML syntax?
  - Init() returns an error describing the syntax issue with line number (never writes to stdout/stderr, even without logger)
  - Error returned to caller for handling; no silent failures
- What happens when environment variable name conflicts with nested config key?
  - Environment variables use underscore-separated uppercase (e.g., `TOOLS_VERSION` for `tools.version`)
- What happens when Mage target receives unexpected named argument?
  - Viper silently ignores unknown flags per standard behavior (projects can add custom validation if needed)
- What happens when config.yaml contains sensitive data?
  - Package redacts sensitive values as "[REDACTED]" in logs via two mechanisms: (1) manual marking of sensitive keys via functional options during Init() (e.g., `config.Init(config.WithSensitiveKeys("api.token", "db.password"))`), (2) automatic detection using gitleaks secret scanning library (proven regex patterns and entropy analysis)
  - File storage responsibility remains with user (use `.gitignore` and external secret management)
- What happens when Init() encounters an error but no logger is provided?
  - Init() returns error for any config-related failure (YAML parsing errors, file access issues, etc.)
  - No output to stdout/stderr when logger not provided (silent operation contract)
  - Caller receives error return value and decides how to handle (fail-fast, fallback, etc.)

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Package MUST provide `Init()` function that loads `config.yaml` from project root and returns error for any config-related failure (never writes to stdout/stderr)
- **FR-002**: Package MUST handle missing `config.yaml` gracefully without error (optional config, missing file is not an error condition)
- **FR-003**: Package MUST support environment variable overrides with standard naming (uppercase, underscore-separated)
- **FR-004**: Package MUST provide `CleanOSArgs()` function to strip non-Mage flags before Viper initialization
- **FR-005**: Package MUST preserve Mage's `-v` verbose flag during `CleanOSArgs()`
- **FR-006**: Package MUST support nested configuration keys accessible via dot notation (e.g., `tools.golangci-lint.version`)
- **FR-007**: Package MUST be importable by external projects without side effects
- **FR-008**: Package MUST enable pflag binding to Viper (using viper.BindPFlags) with documentation showing how to bind named arguments to configuration keys
- **FR-009**: Package MUST return errors with clear context when YAML parsing fails (error return value, no console output without logger)
- **FR-010**: Package MUST support standard Viper features (Get, GetString, GetInt, Unmarshal, etc.)
- **FR-011**: Package MUST accept an optional logger interface with single method signature: `Log(ctx context.Context, level Level, msg string, attrs ...any)` for configuration loading visibility with context propagation (silent if not provided - no stdout/stderr output)
- **FR-012**: Package MUST allow marking configuration keys as sensitive for redaction in logs via functional options pattern during Init() (e.g., `config.Init(config.WithSensitiveKeys("api.token", "db.password"))`)
- **FR-013**: Package MUST automatically detect and redact secrets using gitleaks (`github.com/gitleaks/gitleaks/v8/detect`) detection library (leverage proven regex patterns and entropy analysis instead of simple substring matching)

### Key Entities

- **Configuration**: Represents the loaded config.yaml data structure
  - Attributes: hierarchical key-value pairs, supports strings, numbers, booleans, arrays, maps
  - Source: config.yaml file, environment variables, default values
  - Lifecycle: Loaded once during initialization, immutable after loading

- **Named Argument**: Represents a command-line flag bound to a Viper key
  - Attributes: flag name, Viper key path, default value, type
  - Behavior: Overrides config file values when provided
  - Scope: Per Mage target invocation

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Developers can integrate the package in under 5 minutes (add import, call Init() in magefile init function)
- **SC-002**: Package reduces magefile configuration boilerplate by at least 50% compared to manual Viper setup
- **SC-003**: Zero flag parsing conflicts when using standard Mage commands with Viper-based configuration
- **SC-004**: 100% of Mage targets with named arguments correctly receive parameter values
- **SC-005**: Package works identically across multiple different projects without project-specific code
- **SC-006**: Configuration loading adds less than 10ms overhead to Mage target startup time

## Assumptions

### Technical Assumptions

- Projects using this package have `config.yaml` in their repository root
- Projects are using Mage as their build tool (this is a Mage-specific integration)
- Developers are familiar with basic Viper usage (Get, GetString, etc.)
- Go 1.23+ is available (per constitution requirement)

### Scope Boundaries

**In Scope**:
- Loading config.yaml with standard patterns
- Preventing Mage/Viper flag conflicts
- Supporting named arguments for Mage targets
- Environment variable overrides
- Error handling for common config issues

**Out of Scope**:
- Configuration file validation against schemas (users handle with their own logic)
- Encrypted configuration values (use external secret management)
- Dynamic configuration reloading (config is static after init)
- Alternative config formats (TOML, JSON) - YAML only per existing patterns
- Configuration file generation or scaffolding (manual creation)

### Dependency Assumptions

- `github.com/spf13/viper` package is available and maintained
- Mage framework remains compatible with current flag handling approach
- Projects have write access to `os.Args` during initialization

## Non-Functional Requirements

### Performance

- Configuration loading completes in under 10ms for typical config files (< 100 keys)
- Memory footprint under 1MB for loaded configuration data
- Zero performance impact on Mage target execution after initialization

### Compatibility

- Works on Linux, macOS, and Windows (Go's cross-platform support)
- Compatible with Go 1.23+ (per constitution)
- No platform-specific code or dependencies

### Usability

- API design matches idiomatic Go patterns (lowercase package name, explicit initialization)
- Error messages include actionable guidance (e.g., "config.yaml not found at /path - this is optional, continuing with defaults")
- GoDoc comments provide complete usage examples

### Observability

- Package accepts optional logger via minimal interface: `Log(level Level, msg string, attrs ...any)`
- Interface design: single method for maximum adapter flexibility (slog, logr, zap wrappers trivial)
- Standard logging approach: log errors when they occur, log config source confirmation on success (file path, environment variables used), nothing else
- Silent operation when no logger supplied (default behavior): Init() never writes to stdout/stderr, returns errors via error return value
- Error handling contract: Init() returns error for any config-related failure (YAML parsing, file access, etc.), regardless of logger presence
- Designed for future integration with mage logging library

### Security

- Sensitive configuration values redacted as "[REDACTED]" in log output
- Two-tier protection mechanism:
  1. **Manual marking**: Explicitly mark keys during Init() via functional options pattern (e.g., `config.Init(config.WithSensitiveKeys("api.token", "db.password"))`)
  2. **Automatic detection**: Use gitleaks (`github.com/gitleaks/gitleaks/v8/detect`) secret scanning library
- Auto-detection leverages battle-tested regex patterns and entropy analysis from gitleaks instead of simple substring matching (tokens, keys, credentials, API secrets, etc.)
- Custom secrets not covered by library patterns can be explicitly marked via functional options
- File security remains user responsibility (use `.gitignore`, external secret management)
- No sensitive data written to stdout/stderr (silent operation contract when logger not provided)

## References

- Target projects: See `.private/project-references.md` (internal use only)
- Viper documentation: https://github.com/spf13/viper
- Mage documentation: https://magefile.org/
- Constitution: `.specify/memory/constitution.md` (Principle II: Viper-Based Configuration)
- Secret detection libraries:
  - https://github.com/gitleaks/gitleaks
- Redaction library (implementation reference): https://github.com/cockroachdb/redact

## Revision History

| Date       | Version | Changes                  | Author      |
|------------|---------|--------------------------|-------------|
| 2026-02-14 | 1.0     | Initial specification    | Claude Code |
