# Data Model: Viper Configuration Package

**Feature**: 001-viper-config
**Created**: 2026-02-14
**Phase**: Phase 1 - Design

## Overview

This document defines the core entities, types, and their relationships in the Viper configuration package. The design follows a functional options pattern for initialization with pluggable logger and redaction capabilities.

## Core Entities

### 1. Config (Internal State)

**Purpose**: Holds the runtime configuration state including Viper instance, logger, and redactor.

**Type**: `struct` (unexported)

```go
type config struct {
    viper     *viper.Viper        // Viper instance
    logger    Logger              // Optional logger (nil if not provided)
    redactor  *redact.Redactor    // Value redaction handler
    cleanArgs []string            // Preserved flags from CleanOSArgs
}
```

**Lifecycle**:
- Created via `Init(...Option)` function
- Lives for application lifetime (package-level singleton pattern)
- Immutable after initialization (no setter methods)

**Relationships**:
- Has-one Logger (optional, may be nil)
- Has-one Redactor (always present, may be no-op)
- Owns-one Viper instance (wrapped, not exported)

---

### 2. Option (Functional Option)

**Purpose**: Configures the package during initialization.

**Type**: `func(*config) error`

```go
type Option func(*config) error
```

**Instances**:

```go
// WithLogger sets the optional logger for config loading events
func WithLogger(logger Logger) Option

// WithSensitiveKeys marks config keys as sensitive for redaction
func WithSensitiveKeys(keys ...string) Option

// WithoutSensitiveKeys unmarks config keys as sensitive
func WithoutSensitiveKeys(keys ...string) Option

// WithConfigFile overrides the default config file name (default: "config")
func WithConfigFile(name string) Option

// WithConfigPaths adds additional search paths for config files
func WithConfigPaths(paths ...string) Option

// WithEnvPrefix sets environment variable prefix (default: no prefix)
func WithEnvPrefix(prefix string) Option
```

**Validation**:
- Options return `error` if validation fails
- Examples: nil logger check, invalid path check, empty key name check

**Execution**:
- Applied sequentially during `Init()` call
- Later options can override earlier ones
- Errors stop initialization immediately

---

### 3. Logger (Interface)

**Purpose**: Minimal logging contract for adapter compatibility.

**Type**: `interface`

```go
type Logger interface {
    // Log emits a log message at the specified level with structured attributes
    // ctx enables trace propagation for observability
    // attrs must be key-value pairs (even number of elements)
    Log(ctx context.Context, level Level, msg string, attrs ...any)
}
```

**Level Definition**:

```go
type Level int

const (
    LevelDebug Level = iota  // Detailed internal operations
    LevelInfo                // Standard operational messages
    LevelWarn                // Non-critical issues
    LevelError               // Failures requiring attention
)
```

**Implementations** (user-provided):
- `SlogAdapter` wraps `*slog.Logger`
- `ZerologAdapter` wraps `zerolog.Logger`
- `ZapAdapter` wraps `*zap.Logger`
- Custom implementations for other loggers

**Contract**:
- MUST handle odd-length `attrs` gracefully (ignore last orphan key)
- MUST NOT panic on any input
- MAY buffer/batch log writes
- MAY filter by level (implementation choice)

---

### 4. Redactor (Internal)

**Purpose**: Masks sensitive values in logs.

**Type**: `struct` (internal package `internal/redact`)

```go
type Redactor struct {
    detector      *detect.Detector  // Gitleaks detector
    sensitiveKeys map[string]bool   // Manually marked keys
}
```

**Methods**:

```go
// NewRedactor creates a redactor with optional sensitive keys
func NewRedactor(sensitiveKeys ...string) (*Redactor, error)

// RedactValue returns redacted string for logging
func (r *Redactor) RedactValue(key string, value any) string

// IsSensitive checks if a key should be redacted
func (r *Redactor) IsSensitive(key string) bool
```

**Redaction Rules** (priority order):

1. **Manual keys**: If `key` in `sensitiveKeys` map → `***REDACTED***`
2. **Gitleaks detection**: If value matches secret pattern → `***SECRET_DETECTED:rule-id***`
3. **Default**: Return value as-is (no redaction)

**Performance**:
- Lazy initialization of detector
- Key map lookup: O(1)
- Secret detection: O(n) on value length (cached regex compilation)

---

## Value Types

### ConfigValue (Returned to Users)

**Purpose**: Typed config value accessors via Viper passthrough.

**Access Pattern**:

```go
// Global viper instance (via package functions)
viper.GetString("database.host")       // Returns string
viper.GetInt("server.port")            // Returns int
viper.GetBool("logging.enabled")       // Returns bool
viper.GetStringSlice("allowed.ips")    // Returns []string
viper.Get("custom.setting")            // Returns any
```

**Type Conversion**:
- Handled by Viper (not our responsibility)
- Zero values returned for missing keys
- No panics on type mismatches (returns zero value)

---

## Package-Level Functions

### Query Functions

#### GetSensitiveKeys

```go
func GetSensitiveKeys() []string
```

**Purpose**: Retrieve list of manually marked sensitive keys for debugging and auditing.

**Return Value**:
- `[]string`: Sorted copy of sensitive keys list
- Empty slice if no keys marked
- Does NOT include automatically detected keys (pattern-based)

**Behavior**:
- Returns copy (modifications don't affect internal state)
- Safe to call before or after `Init()`
- Keys are sorted alphabetically for consistent output
- Thread-safe (read-only access to internal state)

**Use Cases**:
- Configuration auditing
- Debugging redaction behavior
- Validation in tests
- Security compliance checks

**Example**:
```go
config.Init(
    config.WithSensitiveKeys("database.password", "api.token"),
    config.WithSensitiveKeys("stripe.secret_key"),
)

keys := config.GetSensitiveKeys()
// keys = ["api.token", "database.password", "stripe.secret_key"]
```

---

## State Transitions

### Initialization States

```
[Uninitialized] --> Init(opts...) --> [Initialized]
                                  ├--> [InitError] (on failure)

States:
- Uninitialized: Default state, global viper not configured
- Initializing: During Init() execution
- Initialized: Config loaded, logger+redactor ready
- InitError: Initialization failed (invalid options, config parse error)
```

**State Checks**:

```go
var (
    initialized bool         // Global flag
    initOnce    sync.Once    // Ensures Init() runs only once
)

func Init(opts ...Option) error {
    var err error
    initOnce.Do(func() {
        err = initializeConfig(opts)
        if err == nil {
            initialized = true
        }
    })
    return err
}
```

**Thread Safety**:
- `Init()` is safe to call concurrently (sync.Once guarantee)
- First call wins, subsequent calls are no-op
- Viper reads are safe after initialization (read-only access)

---

## Validation Rules

### During Initialization

| Field | Validation Rule | Error Message |
|-------|----------------|---------------|
| `WithLogger(logger)` | `logger != nil` | "logger cannot be nil" |
| `WithSensitiveKeys(keys)` | `len(keys) > 0` | "sensitive keys list cannot be empty" |
| `WithSensitiveKeys(keys)` | All keys non-empty strings | "sensitive key cannot be empty string" |
| `WithoutSensitiveKeys(keys)` | `len(keys) > 0` | "keys list cannot be empty" |
| `WithoutSensitiveKeys(keys)` | All keys non-empty strings | "key cannot be empty string" |
| `WithConfigFile(name)` | `name != ""` | "config file name cannot be empty" |
| `WithConfigPaths(paths)` | All paths valid directories | "config path does not exist: %s" |

### During Runtime

| Operation | Rule | Behavior |
|-----------|------|----------|
| `viper.Get*()` | Key exists | Return value |
| `viper.Get*()` | Key missing | Return zero value (no error) |
| `viper.Get*()` | Type mismatch | Return zero value (no error) |
| `logger.Log()` | Logger is nil | No-op (silent) |
| `redactor.RedactValue()` | Always succeeds | Returns string (never errors) |

---

## Relationships & Dependencies

```
[Init(opts)] --> creates --> [config]
                               |
                               +-- uses --> [viper.Viper]
                               +-- has-optional --> [Logger]
                               +-- owns --> [Redactor]
                                               |
                                               +-- uses --> [gitleaks.Detector]
                                               +-- has --> [sensitiveKeys map]

[CleanOSArgs()] --> modifies --> [os.Args]
                --> returns --> [preserved flags]
```

**External Dependencies**:
- `github.com/spf13/viper` - Config management
- `github.com/spf13/pflag` - Flag parsing
- `github.com/zricethezav/gitleaks/v8/detect` - Secret detection

**Internal Dependencies**:
- `internal/redact` - Redaction logic (isolated from main package)

---

## Error Handling

### Error Types

```go
// Public errors (returned to callers)
var (
    ErrAlreadyInitialized = errors.New("config already initialized")
    ErrInvalidOption      = errors.New("invalid initialization option")
    ErrConfigParseFailed  = errors.New("failed to parse config file")
)
```

### Error Propagation

```go
func Init(opts ...Option) error {
    // Apply options (any error stops initialization)
    for _, opt := range opts {
        if err := opt(&cfg); err != nil {
            return fmt.Errorf("%w: %v", ErrInvalidOption, err)
        }
    }

    // Read config (errors returned, not logged)
    if err := cfg.viper.ReadInConfig(); err != nil {
        // Missing file is OK (env vars still work)
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("%w: %v", ErrConfigParseFailed, err)
        }
    }

    return nil
}
```

### Silent Operation Contract

When `Logger` is nil:
- ✅ `Init()` returns errors (never silent failures)
- ✅ No output to stdout/stderr
- ✅ Errors propagated to caller for handling
- ❌ No log messages (silent operation)

---

## Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| `Init()` | O(n·m) | n=config keys, m=env vars |
| `viper.Get*()` | O(1) | Hash map lookup |
| `redactor.RedactValue()` | O(k) | k=value length (regex scan) |
| `logger.Log()` | O(1) | Delegated to implementation |

### Memory Profile

| Component | Size | Lifecycle |
|-----------|------|-----------|
| `config` struct | ~200 bytes | Application lifetime |
| Viper instance | ~10KB + config data | Application lifetime |
| Redactor | ~5KB + rules | Application lifetime |
| Logger | Implementation-dependent | Application lifetime |

### Benchmarks (Target)

```go
BenchmarkInit-8                  100      10ms/op      5MB/op
BenchmarkGetString-8         1000000      100ns/op     0B/op
BenchmarkRedactValue-8       100000       1μs/op       128B/op
BenchmarkLogEvent-8          500000       500ns/op     256B/op
```

---

## Example Usage Flow

```go
package main

import (
    "log/slog"
    "github.com/randomvariable/mage-common/config"
)

func init() {
    // 1. Clean os.Args for Mage compatibility
    config.CleanOSArgs()

    // 2. Initialize with logger and sensitive keys
    logger := config.NewSlogAdapter(slog.Default())
    err := config.Init(
        config.WithLogger(logger),
        config.WithSensitiveKeys("database.password", "api.token"),
    )
    if err != nil {
        panic(err) // Fail fast if config broken
    }
}

func Build() {
    // 3. Use Viper directly (global instance configured)
    platform := viper.GetString("build.platform")
    arch := viper.GetString("build.arch")

    // Values automatically loaded from:
    // - config.yaml (if exists)
    // - Environment variables (e.g., BUILD_PLATFORM, BUILD_ARCH)
    // - Command-line flags (after CleanOSArgs processing)
}
```

---

## Open Questions

None - all design decisions resolved via research phase.

---

## References

- Research Topic 1: Gitleaks for secret detection
- Research Topic 2: Functional options pattern
- Research Topic 3: Logger interface design
- Research Topic 4: Viper integration patterns
- Research Topic 5: Mage flag handling (CleanOSArgs)
- Research Topic 6: Redaction patterns (simple vs cockroachdb/redact)
