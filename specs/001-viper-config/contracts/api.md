# API Contract: Viper Configuration Package

**Feature**: 001-viper-config
**Created**: 2026-02-14
**Phase**: Phase 1 - Design

## Public API Surface

This document defines the complete public API contract for the configuration package. All signatures are guaranteed stable within major versions (semver).

---

## Package: config

**Import Path**: `github.com/randomvariable/mage-common/config`

---

### Initialization Functions

#### Init

```go
func Init(opts ...Option) error
```

**Purpose**: Initialize the global Viper configuration with optional logger, redaction, and settings.

**Parameters**:
- `opts ...Option`: Zero or more configuration options (applied sequentially)

**Returns**:
- `error`: Non-nil if initialization fails (invalid option, config parse error)

**Behavior**:
- Idempotent: Safe to call multiple times (first call wins via `sync.Once`)
- Thread-safe: Concurrent calls are safe
- Searches for `config.{yaml,yml,json,toml}` in current directory
- Binds environment variables automatically (uppercase, underscore-separated)
- Reads command-line flags if `pflag.Parse()` was called before `Init()`

**Errors**:
- `ErrInvalidOption`: Option validation failed (nil logger, empty string, etc.)
- `ErrConfigParseFailed`: config.yaml exists but contains invalid YAML/JSON/TOML
- Note: Missing config file is NOT an error (env vars still work)

**Example**:
```go
err := config.Init(
    config.WithLogger(logger),
    config.WithSensitiveKeys("password", "api_key"),
)
if err != nil {
    log.Fatal(err)
}
```

**Guarantees**:
- ✅ After successful return, `viper.Get*()` methods work correctly
- ✅ Logger receives initialization events if provided
- ✅ Sensitive keys are redacted in all logs
- ✅ Never writes to stdout/stderr (only logs via Logger if provided)

---

#### CleanOSArgs

```go
func CleanOSArgs() []string
```

**Purpose**: Remove long flags (`--flag`) from `os.Args` to prevent Mage conflicts, must be called BEFORE `Init()`.

**Returns**:
- `[]string`: The flags that were removed (for debugging)

**Side Effects**:
- **MODIFIES** `os.Args` in-place
- Preserves short flags (`-v`, `-d`) for Mage
- Preserves non-flag arguments (target names, etc.)

**Timing**:
- MUST call in `init()` function
- MUST call AFTER `pflag.Parse()` (so flags are captured)
- MUST call BEFORE `Init()` (so Mage sees clean args)

**Algorithm**:
```
For each arg in os.Args:
  If starts with "--":
    If contains "=":
      Save to removed list (--flag=value)
    Else:
      Save to removed list along with next arg (--flag value)
  Else:
    Keep in os.Args (binary name, target name, short flags)
```

**Example**:
```go
func init() {
    pflag.String("image", "", "container image name")
    pflag.Parse()

    removed := config.CleanOSArgs()
    // removed = ["--image=webapp"] if called with: mage build --image=webapp

    config.Init()

    // Now os.Args = ["mage", "build"]
    // But viper.GetString("image") = "webapp" (captured by pflag)
}
```

**Guarantees**:
- ✅ After call, `os.Args` contains only binary name, target, and short flags
- ✅ Removed flags are still accessible via Viper (captured by pflag)
- ✅ Mage can parse remaining args without conflicts

---

### Option Functions

#### WithLogger

```go
func WithLogger(logger Logger) Option
```

**Purpose**: Provide a logger for configuration loading events.

**Parameters**:
- `logger Logger`: Implementation of the Logger interface (cannot be nil)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- Returns error if `logger == nil`

**Example**:
```go
logger := config.NewSlogAdapter(slog.Default())
config.Init(config.WithLogger(logger))
```

---

#### WithSensitiveKeys

```go
func WithSensitiveKeys(keys ...string) Option
```

**Purpose**: Mark configuration keys as sensitive for automatic redaction in logs.

**Parameters**:
- `keys ...string`: One or more config keys (dot-notation: `"database.password"`)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- Returns error if `len(keys) == 0`
- Returns error if any key is empty string `""`

**Behavior**:
- Keys are checked case-sensitively
- Nested keys use dot notation: `"api.credentials.token"`
- Multiple calls are cumulative (all keys marked sensitive)

**Example**:
```go
config.Init(
    config.WithSensitiveKeys("database.password", "api.token"),
    config.WithSensitiveKeys("stripe.secret_key"), // Cumulative
)

// All three keys are now redacted in logs
```

---

#### WithoutSensitiveKeys

```go
func WithoutSensitiveKeys(keys ...string) Option
```

**Purpose**: Unmark configuration keys as sensitive, allowing them to appear in logs.

**Parameters**:
- `keys ...string`: One or more config keys to unmark (dot-notation: `"database.password"`)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- Returns error if `len(keys) == 0`
- Returns error if any key is empty string `""`

**Behavior**:
- Removes keys from sensitive keys list
- No-op if key was not marked as sensitive
- Useful for debugging or when sensitivity changes

**Example**:
```go
config.Init(
    config.WithSensitiveKeys("database.password", "api.token"),
    config.WithoutSensitiveKeys("api.token"), // Unmark one key
)

// Only database.password is now redacted
```

---

#### WithConfigFile

```go
func WithConfigFile(name string) Option
```

**Purpose**: Override default config file name (default: `"config"`).

**Parameters**:
- `name string`: File name WITHOUT extension (e.g., `"settings"`, not `"settings.yaml"`)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- Returns error if `name == ""`

**Behavior**:
- Viper searches for `{name}.{yaml,yml,json,toml}` in config paths
- Does not change search paths (use `WithConfigPaths` for that)

**Example**:
```go
config.Init(config.WithConfigFile("settings"))
// Now searches for: settings.yaml, settings.yml, settings.json, settings.toml
```

---

#### WithConfigPaths

```go
func WithConfigPaths(paths ...string) Option
```

**Purpose**: Add additional directories to search for config files (default: `["."]`).

**Parameters**:
- `paths ...string`: One or more directory paths (absolute or relative)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- Returns error if any path does not exist (checked at Init time)

**Behavior**:
- Paths are searched in order (first match wins)
- Default path `"."` is always included (added first)
- Multiple calls are cumulative (all paths added)

**Example**:
```go
config.Init(
    config.WithConfigPaths("/etc/myapp", "$HOME/.config/myapp"),
)
// Search order: ., /etc/myapp, $HOME/.config/myapp
```

---

#### WithEnvPrefix

```go
func WithEnvPrefix(prefix string) Option
```

**Purpose**: Set environment variable prefix for config bindings (default: no prefix).

**Parameters**:
- `prefix string`: Prefix for environment variables (e.g., `"MYAPP"`)

**Returns**:
- `Option`: Configuration option for `Init()`

**Validation**:
- No validation (empty string = no prefix)

**Behavior**:
- All config keys are bound with prefix: `{PREFIX}_{KEY}`
- Key separators (`.`) become underscores (`_`)
- Example: `database.password` → `MYAPP_DATABASE_PASSWORD`

**Example**:
```go
config.Init(config.WithEnvPrefix("MYAPP"))

// Config key: "server.port"
// Environment variable: MYAPP_SERVER_PORT=8080
```

---

### Query Functions

#### GetSensitiveKeys

```go
func GetSensitiveKeys() []string
```

**Purpose**: Retrieve the list of currently marked sensitive keys for debugging.

**Returns**:
- `[]string`: Copy of sensitive keys list (sorted alphabetically)

**Behavior**:
- Returns empty slice if no keys marked
- Returns copy (modifications don't affect internal state)
- Safe to call before or after `Init()`
- Includes only manually marked keys (not automatic pattern matches)

**Example**:
```go
config.Init(
    config.WithSensitiveKeys("database.password", "api.token"),
)

keys := config.GetSensitiveKeys()
// keys = ["api.token", "database.password"] (sorted)
```

**Use Cases**:
- Debugging redaction configuration
- Validation tests
- Configuration auditing

---

### Logger Interface

```go
type Logger interface {
    Log(ctx context.Context, level Level, msg string, attrs ...any)
}
```

**Purpose**: Minimal logging contract for maximum adapter compatibility with context propagation for tracing.

**Context Propagation**:
- `ctx` parameter enables trace ID, span ID, and baggage propagation
- Implementations SHOULD extract trace context from `ctx` if available
- Implementations MAY ignore `ctx` if tracing not needed
- Config package passes `context.Background()` if no context available

**Methods**:

##### Log

```go
Log(ctx context.Context, level Level, msg string, attrs ...any)
```

**Parameters**:
- `ctx context.Context`: Context for cancellation, deadlines, and trace propagation
- `level Level`: Log severity (LevelDebug, LevelInfo, LevelWarn, LevelError)
- `msg string`: Human-readable log message
- `attrs ...any`: Key-value pairs (must be even number of elements)

**Behavior**:
- Accepts variadic key-value pairs: `"key1", value1, "key2", value2`
- Implementations MUST handle odd-length `attrs` gracefully (ignore orphan)
- Implementations MUST NOT panic on any input
- Implementations MAY filter by level

**Example Implementation**:
```go
type SlogAdapter struct {
    logger *slog.Logger
}

func (s *SlogAdapter) Log(ctx context.Context, level Level, msg string, attrs ...any) {
    var slogLevel slog.Level
    switch level {
    case LevelDebug: slogLevel = slog.LevelDebug
    case LevelInfo:  slogLevel = slog.LevelInfo
    case LevelWarn:  slogLevel = slog.LevelWarn
    case LevelError: slogLevel = slog.LevelError
    }
    s.logger.Log(ctx, slogLevel, msg, attrs...)
}
```

---

### Level Type

```go
type Level int
```

**Purpose**: Log severity levels mirroring standard syslog severities.

**Constants**:

```go
const (
    LevelDebug Level = iota // Detailed diagnostic information
    LevelInfo               // General informational messages
    LevelWarn               // Warning messages (non-critical issues)
    LevelError              // Error messages (operation failed)
)
```

**Ordering**:
- `LevelDebug < LevelInfo < LevelWarn < LevelError`
- Numeric values are implementation detail (may change)

**String Representation**:

```go
func (l Level) String() string
```

Returns: `"DEBUG"`, `"INFO"`, `"WARN"`, or `"ERROR"`

---

### Option Type

```go
type Option func(*config) error
```

**Purpose**: Functional option for configuring `Init()`.

**Contract**:
- Returns `nil` on success
- Returns `error` if validation fails
- Applied sequentially during `Init()`
- Errors stop initialization immediately

**User-Defined Options**:

Users can create custom options:

```go
func WithCustomValidator(validator func(string) error) config.Option {
    return func(c *config) error {
        if validator == nil {
            return errors.New("validator cannot be nil")
        }
        c.validator = validator
        return nil
    }
}
```

---

## Adapter Helpers (Recommended Patterns)

### NewSlogAdapter

```go
func NewSlogAdapter(logger *slog.Logger) Logger
```

**Purpose**: Wrap Go 1.21+ slog.Logger for use with config package.

**Example**:
```go
logger := config.NewSlogAdapter(slog.Default())
config.Init(config.WithLogger(logger))
```

---

### NewZerologAdapter

```go
func NewZerologAdapter(logger zerolog.Logger) Logger
```

**Purpose**: Wrap zerolog.Logger for use with config package.

**Example**:
```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
adapter := config.NewZerologAdapter(logger)
config.Init(config.WithLogger(adapter))
```

---

### NewZapAdapter

```go
func NewZapAdapter(logger *zap.Logger) Logger
```

**Purpose**: Wrap zap.Logger for use with config package.

**Example**:
```go
logger, _ := zap.NewProduction()
adapter := config.NewZapAdapter(logger)
config.Init(config.WithLogger(adapter))
```

---

## Logging Behavior

This section defines what events are logged at each level during configuration loading.

### Information Level (LevelInfo)

**Config File Discovery**:
```
msg: "config file loaded"
attrs: "path", "/path/to/config.yaml", "format", "yaml"
```

**Config Summary**:
```
msg: "configuration initialized"
attrs: "keys_loaded", 42, "env_overrides", 3, "sensitive_keys", 2
```

**Environment Overrides**:
```
msg: "environment variable override"
attrs: "key", "database.host", "env_var", "MYAPP_DATABASE_HOST"
```

**Missing Config File (Graceful)**:
```
msg: "config file not found, using defaults and environment"
attrs: "searched_paths", []string{".", "/etc/myapp"}
```

### Warning Level (LevelWarn)

**Flag Stripping**:
```
msg: "removed long flags for Mage compatibility"
attrs: "removed_count", 2, "flags", []string{"--image=webapp", "--tag=latest"}
```

**Deprecated Options** (future use):
```
msg: "deprecated option used"
attrs: "option", "WithOldFeature", "use_instead", "WithNewFeature"
```

### Error Level (LevelError)

**Config Parse Failed**:
```
msg: "failed to parse config file"
attrs: "path", "/path/to/config.yaml", "error", "yaml: line 5: mapping values are not allowed"
```

**Invalid Option**:
```
msg: "initialization option failed"
attrs: "error", "logger cannot be nil", "option", "WithLogger"
```

### Debug Level (LevelDebug)

**Config Loading Steps**:
```
msg: "adding config path"
attrs: "path", "/etc/myapp"

msg: "setting config name"
attrs: "name", "config"

msg: "binding environment variables"
attrs: "prefix", "MYAPP"

msg: "reading config file"
attrs: "attempting", "/etc/myapp/config.yaml"
```

**Viper Binding Events**:
```
msg: "bound flag to config key"
attrs: "flag", "image", "key", "image", "value", "webapp"
```

**Named Argument Resolution**:
```
msg: "resolved named argument"
attrs: "arg", "--image", "key", "image", "value", "webapp", "source", "command-line"
```

**Sensitive Key Detection**:
```
msg: "marking key as sensitive"
attrs: "key", "database.password", "reason", "manual"

msg: "marking key as sensitive"
attrs: "key", "api_token", "reason", "pattern:token"
```

**Configuration State Dump** (debug helper):
```
msg: "configuration state"
attrs: "total_keys", 42, "file_keys", 35, "env_keys", 5, "default_keys", 2,
      "sensitive_keys", ["database.password", "api.token"]
```

---

## Package-Level Variables

### Pre-defined Errors

```go
var (
    ErrAlreadyInitialized = errors.New("config already initialized")
    ErrInvalidOption      = errors.New("invalid initialization option")
    ErrConfigParseFailed  = errors.New("failed to parse config file")
)
```

**Purpose**: Sentinel errors for error handling.

**Example**:
```go
if err := config.Init(); err != nil {
    if errors.Is(err, config.ErrConfigParseFailed) {
        log.Fatal("Fix your config.yaml syntax")
    }
}
```

---

## Viper Access Pattern

After `Init()`, use Viper's global instance directly:

```go
import "github.com/spf13/viper"

// String values
host := viper.GetString("database.host")

// Numeric values
port := viper.GetInt("server.port")
timeout := viper.GetDuration("client.timeout")

// Boolean values
enabled := viper.GetBool("logging.enabled")

// Collections
allowedIPs := viper.GetStringSlice("security.allowed_ips")

// Nested maps
dbConfig := viper.GetStringMap("database")

// Generic (returns any)
customSetting := viper.Get("custom.setting")
```

**Note**: This package configures Viper, then delegates all value access to Viper's API. No wrapper functions needed.

---

## Thread Safety Guarantees

### Init()
- ✅ Safe to call concurrently
- ✅ First call wins (sync.Once)
- ✅ Subsequent calls are no-op (return nil if first succeeded, or original error)

### CleanOSArgs()
- ⚠️ NOT safe to call concurrently (modifies global `os.Args`)
- ✅ Safe if called in `init()` (single-threaded execution)

### viper.Get*()
- ✅ Safe to call concurrently (read-only after Init())
- ✅ No locks needed for reads

### Logger.Log()
- ⚠️ Thread-safety is implementation-dependent
- Standard adapters (slog, zerolog, zap) are all thread-safe

---

## Breaking Change Policy

**Major Version (v2.x.x)**:
- Signature changes (parameters, return types)
- Removal of exported functions/types
- Behavioral changes that break existing usage

**Minor Version (v1.x.x)**:
- New exported functions/types
- New optional parameters (via functional options)
- Backward-compatible enhancements

**Patch Version (v1.0.x)**:
- Bug fixes
- Performance improvements
- Documentation updates

---

## Examples

### Minimal Usage

```go
package main

import (
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/viper"
)

func init() {
    config.CleanOSArgs()
    config.Init() // No options = silent mode
}

func Build() {
    platform := viper.GetString("build.platform")
    // Use config values...
}
```

---

### Full-Featured Usage

```go
package main

import (
    "log/slog"
    "os"
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func init() {
    // 1. Define flags
    pflag.String("image", "", "container image")
    pflag.String("tag", "latest", "image tag")
    pflag.Parse()

    // 2. Clean os.Args for Mage
    config.CleanOSArgs()

    // 3. Setup logger
    logger := config.NewSlogAdapter(
        slog.New(slog.NewJSONHandler(os.Stdout, nil)),
    )

    // 4. Initialize config
    err := config.Init(
        config.WithLogger(logger),
        config.WithSensitiveKeys("registry.password", "api.token"),
        config.WithConfigFile("build"),
        config.WithConfigPaths(".", "/etc/myapp"),
        config.WithEnvPrefix("MYAPP"),
    )
    if err != nil {
        panic(err)
    }
}

func Build() {
    // Precedence order (highest to lowest):
    // 1. Command-line: mage build --image=webapp
    // 2. Environment:   MYAPP_IMAGE=webapp
    // 3. Config file:   image: webapp (in build.yaml)
    // 4. Default:       "" (from pflag definition)

    image := viper.GetString("image")
    tag := viper.GetString("tag")

    // Sensitive values are redacted in logs:
    // Log output: "config_value key=registry.password value=***REDACTED***"
}
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | TBD | Initial release |

---

## References

- Viper documentation: https://github.com/spf13/viper
- Functional options pattern: research.md Topic 2
- Logger interface design: research.md Topic 3
- CleanOSArgs implementation: research.md Topic 5
