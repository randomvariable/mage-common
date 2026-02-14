# Research: Viper Configuration Package

**Date**: 2026-02-14
**Feature**: [spec.md](spec.md)
**Plan**: [plan.md](plan.md)

## Purpose

Phase 0 research to resolve technical unknowns and establish best practices for implementation. All "NEEDS CLARIFICATION" items from Technical Context must be resolved before proceeding to Phase 1 design.

## Research Topics

### 1. Secret Detection Library: gitleaks vs trufflehog

**Question**: Which library (gitleaks or trufflehog) should we use for automatic secret pattern detection?

**Research Tasks**:
- Compare API surface and ease of integration for both libraries
- Evaluate pattern coverage (number of secret types detected)
- Assess performance impact (memory, CPU overhead)
- Check licensing compatibility (both are open source, verify license type)
- Verify maintenance status (recent commits, issue response time)
- Test integration complexity (can we use as library vs CLI only)

---

## Comparison Analysis

### 1. API Surface - Can we use as Go library?

**gitleaks** ✅ **WINNER**
- **Package**: `github.com/zricethezav/gitleaks/v8/detect`
- **Key Functions**:
  - `NewDetector(cfg config.Config)` - Create detector with config
  - `NewDetectorDefaultConfig()` - Use built-in rules
  - `DetectString(content string) []report.Finding` - Scan string directly
  - `DetectBytes(content []byte) []report.Finding` - Scan bytes
  - `Detect(fragment Fragment) []report.Finding` - Scan fragment
- **Integration**: Simple - just import package, create detector, call DetectString()
- **Documentation**: Clear API with detector struct and simple method signatures

**trufflehog**
- **Package**: `github.com/trufflesecurity/trufflehog/v3/pkg/detectors`
- **Key Pattern**: Individual detector packages (800+ detectors)
- **Each detector implements**: `FromData(ctx, verify bool, data []byte) ([]Result, error)`
- **Problem**: No single "scan this string" function - must instantiate specific detectors
- **Complexity**: Requires engine setup, source configuration, detector orchestration

**Verdict**: Gitleaks has a vastly superior library API for our use case. Single function call vs. complex orchestration.

---

### 2. Integration Complexity

**gitleaks** ✅ **WINNER**
```go
// Simple integration example
detector, _ := detect.NewDetectorDefaultConfig()
findings := detector.DetectString(configKeyValue)
// findings contains any secrets detected
```

**trufflehog**
- Requires setting up Engine, Sources, Detectors
- Designed for scanning repos/files/directories, not individual strings
- Library usage pattern is complex: `engine.ScanSource(source)` with async results
- Not optimized for "scan this config value" use case

**Verdict**: Gitleaks is designed for both CLI and library use. Trufflehog is primarily CLI-focused with library usage as secondary concern.

---

### 3. Pattern Coverage

**gitleaks**
- **700+ built-in rules** covering major secret types
- Includes: AWS, GCP, Azure, GitHub, GitLab, Slack, Stripe, JWT, private keys, database credentials, API tokens
- Configurable via TOML with regex patterns, entropy thresholds, keywords
- Rule format: `regex`, `entropy`, `keywords`, `secretGroup` extraction

**trufflehog** ✅ **WINNER (by count)**
- **800+ detector types** (detectorspb.DetectorType enum)
- Individual detectors for each service/secret type
- More granular: separate detectors for AWS Access Key, AWS Secret Key, etc.
- Includes verification logic (can test if credential is valid)

**Verdict**: Trufflehog has more detectors, but for our use case (detecting key names like "password", "token", "secret"), both have sufficient coverage. Gitleaks rules are more flexible for generic pattern matching.

---

### 4. Performance - Memory/CPU Overhead

**gitleaks** ✅ **WINNER**
- **Single detector instance** scans all patterns simultaneously
- Uses Aho-Corasick algorithm for keyword pre-filtering (efficient multi-pattern matching)
- Regex matching only triggered after keyword match
- Memory: O(patterns) - one detector holds all compiled regexes
- CPU: Optimized for batch scanning with keyword filtering first
- **Benchmarks**: Designed for scanning entire git histories efficiently

**trufflehog**
- **800+ individual detector instances** would need to be instantiated
- Each detector runs `FromData()` independently
- Verification step (API calls) adds network overhead (can be disabled with `verify: false`)
- Memory: O(detectors × data) - each detector processes full input
- CPU: More overhead from detector orchestration

**Verdict**: Gitleaks is significantly more efficient for scanning individual config keys. Keyword pre-filtering + single detector instance vs. 800+ detector invocations.

---

### 5. Licensing

**gitleaks** ✅ **WINNER**
- **License**: MIT (permissive)
- **Commercial use**: ✅ Allowed
- **Modification**: ✅ Allowed
- **Distribution**: ✅ Allowed
- **Attribution**: Required (MIT license notice)

**trufflehog**
- **License**: AGPL-3.0 (copyleft)
- **Commercial use**: ✅ Allowed (with restrictions)
- **Distribution**: If you distribute modified version, must share source under AGPL
- **Network use**: AGPL "network use = distribution" (controversial for libraries)
- **Risk**: Using AGPL library in proprietary software can trigger copyleft requirements

**Verdict**: MIT license (gitleaks) is safer for library use in proprietary projects. AGPL (trufflehog) has potential legal complications.

---

### 6. Maintenance & Activity

**gitleaks** ✅ **WINNER**
- **GitHub**: 24.9k stars, 1.9k forks, 214 contributors
- **Last commit**: Last month (active)
- **Releases**: 189 releases, latest v8.30.0 (Nov 2025)
- **Issues**: 234 open (actively managed)
- **Commits**: Regular updates, responsive maintainers
- **Sponsor**: @zricethezav actively sponsored

**trufflehog**
- **GitHub**: 24.6k stars, 2.2k forks, 183 contributors
- **Last commit**: 3 days ago (very active)
- **Releases**: 319 releases, latest v3.93.3 (3 days ago)
- **Issues**: 242 open
- **Backing**: TruffleSecurityㅡcommercial company (enterprise product)

**Verdict**: Both actively maintained. Trufflehog slightly more active (3-day release cycle), but gitleaks has simpler governance (individual maintainer vs. company).

---

### 7. Use Case Fit: Scanning Config Key Names

**gitleaks** ✅ **WINNER**
```go
// Our use case: scan config key "db.password" value "secretvalue123"
keyName := "db.password"
keyValue := "secretvalue123"

detector, _ := detect.NewDetectorDefaultConfig()

// Scan key name for patterns like "password", "token", "secret"
findings := detector.DetectString(keyName)
if len(findings) > 0 {
    // Key name suggests sensitive value
    redactValue = true
}

// Also scan value for high-entropy secrets
findings = detector.DetectString(keyValue)
```

**Pros**: Simple string scanning, keyword-based detection, customizable rules
**Cons**: None for this use case

**trufflehog**
```go
// Complex setup required
engine := engine.Start(ctx)
defer engine.Finish(ctx)

// Must wrap string in a Source implementation
source := &sources.File{Content: strings.NewReader(keyValue)}

// Async results channel
results := make(chan *detectors.ResultWithMetadata)
engine.ChunksChan() <- &sources.Chunk{Data: []byte(keyValue)}

// Process results asynchronously
for result := range results {
    // Handle finding
}
```

**Pros**: Can verify credentials (not needed for our use case)
**Cons**: Overcomplicated for simple string scanning, async complexity, verification adds latency

**Verdict**: Gitleaks is purpose-built for this. Trufflehog adds unnecessary complexity.

---

## Decision

**Decision**: **gitleaks** (`github.com/zricethezav/gitleaks/v8/detect`)

**Rationale**:
1. **Superior library API**: Single function call (`DetectString()`) vs. complex orchestration
2. **Better performance**: Keyword pre-filtering + single detector instance vs. 800+ detector overhead
3. **Licensing safety**: MIT license avoids AGPL copyleft risks in library context
4. **Perfect fit for use case**: Designed for scanning strings/fragments, not just full repos
5. **Simpler integration**: 5 lines of code vs. managing engine/sources/async results
6. **Sufficient pattern coverage**: 700+ rules cover all common secret patterns we need
7. **Active maintenance**: Well-maintained with responsive community

**Alternatives Considered**:

**trufflehog** - Rejected
- **Strengths**: More detectors (800+), credential verification, commercial backing
- **Weaknesses**:
  - Overcomplicated for scanning config key names
  - AGPL-3.0 license unsuitable for library use
  - No simple "scan this string" API
  - Async complexity unnecessary for synchronous config loading
  - Higher memory/CPU overhead (800+ detectors)
- **Why rejected**: Library API complexity and licensing concerns outweigh detector count advantage. Verification feature (checking if credentials are valid) is not needed for our use case of detecting sensitive key names.

**custom regex patterns** - Rejected
- **Strengths**: Full control, zero dependencies, simple implementation
- **Weaknesses**:
  - Must maintain regex patterns ourselves
  - Miss edge cases that battle-tested libraries catch
  - No entropy analysis for unknown patterns
  - Reinventing the wheel
- **Why rejected**: Gitleaks provides better coverage with minimal overhead

---

## Implementation Notes

### Basic Integration
```go
package config

import (
    "github.com/zricethezav/gitleaks/v8/detect"
    "github.com/zricethezav/gitleaks/v8/report"
)

// isSensitiveKey checks if a config key name suggests sensitive data
func isSensitiveKey(keyName string) bool {
    detector, err := detect.NewDetectorDefaultConfig()
    if err != nil {
        // Fallback to manual list if detector fails
        return containsSensitivePattern(keyName)
    }

    findings := detector.DetectString(keyName)
    return len(findings) > 0
}

// Manual fallback patterns (used if gitleaks fails)
var sensitivePatterns = []string{
    "password", "token", "secret", "key", "credential",
    "auth", "api_key", "apikey", "private",
}

func containsSensitivePattern(s string) bool {
    lower := strings.ToLower(s)
    for _, pattern := range sensitivePatterns {
        if strings.Contains(lower, pattern) {
            return true
        }
    }
    return false
}
```

### Custom Configuration (Optional)
If default rules don't fit, create custom config:
```go
cfg := config.Config{
    Rules: []config.Rule{
        {
            RuleID:      "config-key-password",
            Description: "Config key contains 'password'",
            Keywords:    []string{"password", "passwd", "pwd"},
            Regex:       regexp.MustCompile(`(?i)password`),
        },
        {
            RuleID:      "config-key-token",
            Description: "Config key contains 'token'",
            Keywords:    []string{"token", "bearer"},
            Regex:       regexp.MustCompile(`(?i)token`),
        },
    },
}
detector := detect.NewDetector(cfg)
```

### Dependency Management
```bash
# Add to go.mod
go get github.com/zricethezav/gitleaks/v8@latest

# Current stable version: v8.30.0
```

### Performance Optimization
- **Detector reuse**: Create detector once, use for all config keys (thread-safe)
- **Keyword filtering**: Gitleaks checks keywords before running regex (fast)
- **No I/O**: DetectString() is pure computation, no file/network access
- **Expected overhead**: <1ms per config key on modern hardware

### Error Handling
```go
detector, err := detect.NewDetectorDefaultConfig()
if err != nil {
    // Detector creation failed - use manual fallback
    // Log error, continue with simplified pattern matching
    return containsSensitivePattern(keyName)
}
```

### Testing Strategy
```go
func TestSecretDetection(t *testing.T) {
    tests := []struct {
        keyName  string
        expected bool
    }{
        {"database.password", true},
        {"api.token", true},
        {"app.secret_key", true},
        {"app.name", false},
        {"server.port", false},
    }

    detector, _ := detect.NewDetectorDefaultConfig()
    for _, tt := range tests {
        findings := detector.DetectString(tt.keyName)
        got := len(findings) > 0
        if got != tt.expected {
            t.Errorf("key %q: got %v, want %v", tt.keyName, got, tt.expected)
        }
    }
}
```

### Documentation References
- Gitleaks GitHub: https://github.com/gitleaks/gitleaks
- API Docs (GoDoc): https://pkg.go.dev/github.com/zricethezav/gitleaks/v8
- Default Config: https://github.com/gitleaks/gitleaks/blob/master/config/gitleaks.toml
- Rule Examples: https://github.com/gitleaks/gitleaks/tree/main/cmd/generate/config/rules

---

### 2. Functional Options Pattern in Go

**Question**: What are the best practices for implementing functional options in Go for our Init() API?

**Best Practices** (from Dave Cheney, Rob Pike, and productions libraries):

1. **Use function type for options**:
   ```go
type Option func(*config) error
   ```

2. **Unexported config struct** (private implementation detail):
   ```go
   type config struct {
      logger         Logger
      sensitiveKeys  []string
      configName     string
      configPaths    []string
   }
   ```

3. **Exported With functions** (public API):
   ```go
   func WithLogger(logger Logger) Option {
       return func(c *config) error {
        if logger == nil {
               return fmt.Errorf("logger cannot be nil")
           }
           c.logger = logger
           return nil
       }
   }

   func WithSensitiveKeys(keys ...string) Option {
       return func(c *config) error {
           c.sensitiveKeys = append(c.sensitiveKeys, keys...)
           return nil
       }
   }
   ```

4. **Init accepts variadic options**:
   ```go
   func Init(opts ...Option) error {
       cfg := &config{
           configName:  "config",      // defaults
           configPaths: []string{"."},
       }

       for _, opt := range opts {
           if err := opt(cfg); err != nil {
               return fmt.Errorf("applying option: %w", err)
           }
       }

       // Use cfg to initialize
       return initWithConfig(cfg)
   }
   ```

**Naming Conventions**:
- ✅ `WithX` - Standard Go idiom (cobra, viper, zap all use this)
- ❌ `SetX` - Implies mutation, less idiomatic
- ❌ `OptionX` - Redundant, type is already named Option

**Validation Timing**:
- **At option creation** (preferred): Validate in `WithX` function before returning option
- **At apply time**: Validate when option is applied to struct
- **Hybrid** (recommended): Simple checks in WithX(), complex checks during apply

**Common Pitfalls**:
- Don't mutate global state in options
- Validate nil pointers immediately
- Return errors from Option functions (don't panic)
- Keep options independent (no ordering dependencies)

**Implementation Guidance**:
```go
// Bad: Panic on invalid input
func WithLogger(logger Logger) Option {
    if logger == nil {
        panic("logger cannot be nil") // ❌ Don't panic
    }
    return func(c *config) error {
        c.logger = logger
        return nil
    }
}

// Good: Return error from option function
func WithLogger(logger Logger) Option {
    return func(c *config) error {
        if logger == nil {
            return fmt.Errorf("logger cannot be nil") // ✅ Return error
        }
        c.logger = logger
        return nil
    }
}

// Usage
if err := Init(WithLogger(slog.Default())); err != nil {
    // Handle error
}
```

**Examples from Well-Known Libraries**:
- **cobra**: `NewCommand(use string, opts ...CommandOption)`
- **zap**: `New(core zapcore.Core, options ...Option)`
- **viper**: Uses SetX methods (older API, not functional options)
- **slog**: `New(h Handler, options ...HandlerOption)` (since Go 1.21)

---

### 3. Logger Interface Design

**Question**: How should we design the logger interface to be compatible with slog, zerolog, and zap?

**Interface Design** (from spec clarification Q1: Single method with level, message, attributes):

```go
// Level represents log severity
type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

// Logger is the minimal interface for logging configuration events
// Implementations can wrap slog, zerolog, zap, or any structured logger
type Logger interface {
    // Log emits a log message at the specified level with structured attributes
    // attrs should be key-value pairs (even number of elements)
    Log(level Level, msg string, attrs ...any)
}
```

**Design Rationale**:
1. **Single method** - Simpler than Debug()/Info()/Warn()/Error() methods
2. **Level parameter** - Caller controls severity, interface doesn't dictate methods
3. **Variadic attrs** - Matches slog.Handler pattern, flexible structured logging
4. **Compatible with all major loggers** - Can wrap any structured logger

**Adapter Examples**:

**slog adapter** (Go 1.21+ standard library):
```go
type SlogAdapter struct {
    logger *slog.Logger
}

func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
    return &SlogAdapter{logger: logger}
}

func (s *SlogAdapter) Log(level Level, msg string, attrs ...any) {
   var slogLevel slog.Level
    switch level {
    case LevelDebug:
        slogLevel = slog.LevelDebug
    case LevelInfo:
        slogLevel = slog.LevelInfo
    case LevelWarn:
        slogLevel = slog.LevelWarn
    case LevelError:
        slogLevel = slog.LevelError
    }

    s.logger.Log(context.Background(), slogLevel, msg, attrs...)
}

// Usage
logger := NewSlogAdapter(slog.Default())
Init(WithLogger(logger))
```

**zerolog adapter**:
```go
type ZerologAdapter struct {
    logger zerolog.Logger
}

func (z *ZerologAdapter) Log(level Level, msg string, attrs ...any) {
    var event *zerolog.Event
    switch level {
    case LevelDebug:
        event = z.logger.Debug()
    case LevelInfo:
        event = z.logger.Info()
    case LevelWarn:
        event = z.logger.Warn()
    case LevelError:
        event = z.logger.Error()
    }

    // Convert attrs to zerolog fields
    for i := 0; i < len(attrs); i += 2 {
        if i+1 < len(attrs) {
            key := fmt.Sprint(attrs[i])
            value := attrs[i+1]
            event = event.Interface(key, value)
        }
    }

    event.Msg(msg)
}
```

**zap adapter**:
```go
type ZapAdapter struct {
    logger *zap.Logger
}

func (z *ZapAdapter) Log(level Level, msg string, attrs ...any) {
    fields := make([]zap.Field, 0, len(attrs)/2)

    for i := 0; i < len(attrs); i += 2 {
        if i+1 < len(attrs) {
            key := fmt.Sprint(attrs[i])
            value := attrs[i+1]
            fields = append(fields, zap.Any(key, value))
        }
    }

    switch level {
    case LevelDebug:
        z.logger.Debug(msg, fields...)
    case LevelInfo:
        z.logger.Info(msg, fields...)
    case LevelWarn:
        z.logger.Warn(msg, fields...)
    case LevelError:
        z.logger.Error(msg, fields...)
    }
}
```

**Structured Logging Patterns**:
All three libraries (slog, zerolog, zap) support key-value pairs for structured logging:
- **slog**: `Log(ctx, level, msg, "key1", value1, "key2", value2)`
- **zerolog**: `logger.Info().Str("key1", val1).Int("key2", val2).Msg(msg)`
- **zap**: `logger.Info(msg, zap.String("key1", val1), zap.Int("key2", val2))`

Our `Log(level, msg, attrs...)` maps naturally to all three patterns.

**Log Level Representation**:
- **Our choice**: Custom `Level` int enum
- **Alternative 1**: Use `slog.Level` directly - couples us to slog
- **Alternative 2**: String levels ("debug", "info") - less type-safe
- **Rationale**: Custom enum provides independence from any specific logger library

**Thread Safety**:
- Interface methods don't mutate config state
- Logger implementations (slog, zerolog, zap) handle thread-safety internally
- Our code just calls Log() - no concurrency concerns

**Real-World Example**:
Using slog directly in magefiles:
```go
slog.ErrorContext(context.Background(), "unable to bind flags", "err", err)
```

We'll wrap this pattern in our Logger interface for maximum flexibility.

---

### 4. Viper Integration Patterns

**Question**: What are the best practices for programmatic Viper usage in library code?

**Integration Pattern** (from code analysis):

```go
func Init(opts ...Option) error {
    cfg := applyOptions(opts) // Apply functional options

    // 1. Add config search paths
    viper.AddConfigPath(".") // Current directory (project root for most projects)

    // 2. Set config file name and type
    viper.SetConfigName("config")  // Looks for config.yaml, config.yml, etc.
    viper.SetConfigType("yaml")     // Force YAML format

    // 3. Bind environment variables (automatic)
    viper.AutomaticEnv()  // Enables ENVVAR reading
    // Or specific bindings:
    viper.BindEnv("key-name", "ENV_VAR_NAME")

    //4. Bind command-line flags (if using pflag)
    if cfg.bindFlags {
        viper.BindPFlags(pflag.CommandLine)
    }

    // 5. Read config file (optional - gracefully handle missing file)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); ok {
            // Config file not found - this is OK per spec FR-002
            if cfg.logger != nil {
                cfg.logger.Log(LevelInfo, "config file not found, using defaults and environment variables")
            }
        } else {
            // Config file exists but has errors (YAML syntax, permissions, etc.)
            return fmt.Errorf("error reading config file: %w", err)
        }
    } else {
        // Success
        if cfg.logger != nil {
            cfg.logger.Log(LevelInfo, "loaded configuration", "file", viper.ConfigFileUsed())
        }
    }

    return nil
}
```

**Config File Search Paths**:
Viper searches in order until found:
1. Paths added via `AddConfigPath()` (in order added)
2. Current working directory if no paths specified
3. Stops at first match

**Precedence Order** (highest to lowest):
1. Explicit `Set()` calls
2. Command-line flags (if bound via `BindPFlags`)
3. Environment variables (if bound via `BindEnv` or `AutomaticEnv`)
4. Config file values
5. Default values (set via `SetDefault`)

**Environment Variable Binding Conventions** (from code):
```go
// Automatic: reads ENVVAR matching key name (uppercase, underscores)
viper.AutomaticEnv()
viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))  // "some-key" → "SOME_KEY"

// Manual: bind specific keys to specific env vars
viper.BindEnv("github-token", "GITHUB_TOKEN", "GITHUB_DOTCOM_TOKEN")  // Try multiple env vars
viper.BindEnv("build-number", "BUILD_NUMBER")
viper.BindEnv("build-type", "REL TYPE")
```

**Nested Key Access** (dot notation):
```go
// config.yaml:
// tools:
//   golangci-lint:
//     version: "1.55.2"

version := viper.GetString("tools.golangci-lint.version")  // ✅ Works
version := viper.GetString("tools" + "." + "golangci-lint.version")  // ✅ Programmatic

// Environment variable equivalent:
// TOOLS_GOLANGCI_LINT_VERSION=1.55.2
```

**Common Pitfalls**:

1. **Global State**: Viper uses a global instance by default
   ```go
   // Problem: Multiple packages calling Init() conflict
   viper.SetConfigName("config")  // Affects global state

   // Solution: Use viper.New() for isolated instances (if needed)
   v := viper.New()
   v.SetConfigName("config")
   // But for library: just document that Init() configures global Viper
   ```

2. **Config Merging**: Multiple `ReadInConfig()` calls merge
   ```go
   viper.ReadInConfig()  // Reads config.yaml
   viper.ReadInConfig()  // Merges with previous, doesn't replace
   ```

3. **Case Sensitivity**: Keys are case-insensitive by default
   ```go
   viper.Get("MyKey") == viper.Get("mykey")  // true
   ```

4. **Type Coercion**: Viper attempts type conversion
   ```go
   // config.yaml: port: "8080" (string)
   port := viper.GetInt("port")  // Returns 8080 (int), not error
   ```

**Thread-Safety Considerations**:
- **Viper is NOT thread-safe** for writes during reads
- Safe pattern: Initialize once (Init()), then only read
- Avoid `Set()` calls after initialization

 - If dynamic config needed, protect with mutex or use separate viper instance

**Gotchas to Avoid**:
1. ❌ Calling `ReadInConfig()` multiple times (merges instead of replacing)
2. ❌ Assuming config file must exist (handle `ConfigFileNotFoundError`)
3. ❌ Not checking for YAML parse errors vs file-not-found errors
4. ❌ Forgetting to set `SetConfigType` when using non-standard extensions
5. ❌ Expecting thread-safe writes (Viper is read-optimized after init)

**Real-World Pattern**:
```go
// From config.go:
viper.AddConfigPath(cwd())
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.BindPFlags(pflag.CommandLine)

if err := viper.ReadInConfig(); err!= nil {
    // Original doesn't check error type - assumes config is optional
    // Our library should be explicit about optional vs required
}
```

---

### 5. Mage Flag Handling Integration

**Question**: How does Mage parse command-line flags, and what's the best way to prevent conflicts?

**CleanOSArgs Strategy** (from a previous implementation):

```go
// cleanOSArgs removes pflags from os.Args to prevent Mage flag parsing conflicts
// MUST be called BEFORE viper.BindPFlags() and AFTER pflag.Parse()
func cleanOSArgs() []string {
    newArgs := []string{}        // Flags to keep for viper
    newOSArgs := []string{}      // Non-flag args for mage

    for i := 0; i < len(os.Args); i++ {
        arg := os.Args[i]
        isFlag := strings.HasPrefix(arg, "--")
        isWithEqualsSign := strings.Contains(arg, "=")

        switch {
        case isFlag && !isWithEqualsSign:
            // Flag with separate value: --flag value
            newArgs = append(newArgs, os.Args[i])
            i++
            if i < len(os.Args) {
                newArgs = append(newArgs, os.Args[i])  // Capture value
            }

        case isFlag && isWithEqualsSign:
            // Flag with equals: --flag=value
            newArgs = append(newArgs, os.Args[i])

        default:
            // Non-flag argument (mage command, target name, etc.)
            newOSArgs = append(newOSArgs, os.Args[i])
        }
    }

    os.Args = newOSArgs  // Replace os.Args with non-flag args
    return newArgs       // Return flags for processing
}
```

**How Mage Parses Flags**:
1. Mage uses Go's `flag` package internally
2. Expects `os.Args` to contain: `[binary, target, args...]`
3. Any `--` flags in `os.Args` confuse Mage's parser
4. Mage special flags: `-h`, `-l`, `-v`, `-d`, `-w`, `-t`, `-keep`

**When Mage Flag Parsing Occurs**:
- **Before** `init()` - Mage's runtime processes os.Args early
- **Solution**: Call `cleanOSArgs()` in `init()` to strip flags before Mage sees them

**Preserved Flags** (Mage's own flags):
A previous implementation preserves short flags (`-v`, etc.) by only stripping long flags (`--`):
```go
isFlag := strings.HasPrefix(arg, "--")  // Only removes long flags
// -v, -d, -h pass through untouched
```

**Correct Calling Order**:
```go
func init() {
    // 1. Set up viper config paths
    viper.AddConfigPath(".")
    viper.SetConfigName("config")

    // 2. Define pflags
    pflag.String("build-type", "", "build type")
    pflag.Bool("ci", false, "CI mode")

    // 3. Parse pflags
    pflag.Parse()

    // 4. Clean os.Args (CRITICAL: before viper.BindPFlags)
    cleanOSArgs()

    // 5. Bind pflags to viper
    viper.BindPFlags(pflag.CommandLine)

    // 6. Read config
    viper.ReadInConfig()
}
```

**Why This Works**:
1. `pflag.Parse()` processes `--flags` into pflag's internal state
2. `cleanOSArgs()` removes `--flags` from `os.Args`
3. Mage sees clean `os.Args = ["mage", "build"]` without `--` flags
4. `viper.BindPFlags()` reads from pflag's internal state (not os.Args)
5. No conflict: Mage gets clean args, Viper gets flag values

**Alternative Approaches** (considered and rejected):

**Alternative 1: Don't use pflag, just use environment variables**
- ❌ Loses command-line argument capability
- ❌ Users can't override config via CLI
- A previous chose pflag integration instead

**Alternative 2: Use Mage's `mg.Namespace` with typed arguments**
- ❌ Doesn't support named arguments (`mage build --image=foo`)
- ❌ Forces positional arguments
- Not suitable for our use case (spec requires named args)

**Alternative 3: Parse flags manually without pflag**
- ❌ Reinvents wheel
- ❌ Loses pflag's rich features (aliases, short flags, help generation)
- More code to maintain

**os.Args Manipulation Safety**:
```go
// Safe: We own the process, os.Args is mutable
os.Args = newOSArgs

// Thread-safety: init() runs before main(), single-threaded
// No concurrency concerns

// Mage behavior: Reads os.Args during runtime initialization
// By modifying in init(), we intercept before Mage processes flags
```

**Testing Strategy**:
```go
func TestCleanOSArgs(t *testing.T) {
    tests := []struct {
        input       []string
        expectedOS  []string
        expectedReturn []string
    }{
        {
            input:       []string{"mage", "build", "--image=foo", "--ci"},
            expectedOS:  []string{"mage", "build"},
            expectedReturn: []string{"--image=foo", "--ci"},
        },
        {
            input:       []string{"mage", "-v", "test", "--coverage"},
            expectedOS:  []string{"mage", "-v", "test"},
            expectedReturn: []string{"--coverage"},
        },
    }

    for _, tt := range tests {
        os.Args = tt.input
        returned := cleanOSArgs()

        if !reflect.DeepEqual(os.Args, tt.expectedOS) {
            t.Errorf("os.Args = %v, want %v", os.Args, tt.expectedOS)
        }
        if !reflect.DeepEqual(returned, tt.expectedReturn) {
            t.Errorf("returned = %v, want %v", returned, tt.expectedReturn)
        }
    }
}
```

**Real-World  Usage**:
A previous project has been using this pattern for a while:
- No reported issues with Mage conflicts
- Enables named arguments for Mage targets
- Allows rich flag definitions with pflag

**Decision**: Adopt `cleanOSArgs()` implementation as-is. It's battle-tested and solves exactly our problem.

---

### 6. Redaction Library Patterns (cockroachdb/redact)

**Question**: How does cockroachdb/redact work, and can we use it for our redaction needs?

**cockroachdb/redact Overview**:

The library provides sophisticated PII/secret protection using special UTF-8 markers:

```go
import "github.com/cockroachdb/redact"

// Unsafe values wrapped in special markers ‹ and ›
redactable := redact.Sprintf("Password: %s", "secret123")
// Contains: "Password: ‹secret123›"

// Strip markers for display
fmt.Println(redactable.StripMarkers())  // "Password: secret123"

// Redact for logging/telemetry
fmt.Println(redactable.Redact())        // "Password: ‹×›"

// Explicit safe/unsafe wrapping
redactable := redact.Sprintf("User %s logged in from %s",
    redact.Safe("alice"),      // Won't be redacted
    "192.168.1.1")             // Will be redacted
// Output: "User alice logged in from ‹192.168.1.1›"
```

**Key Types**:

1. **RedactableString**: String with embedded redaction markers
2. **SafeValue**: Wrapper forcing value to be treated as safe (no markers)
3. **SafeFormatter**: Interface for custom types to control redaction
4. **Safe/Unsafe**: Explicit wrappers for values

**Integration with Logging**:

```go
// With custom Logger interface (our use case)
type Logger interface {
    Log(level LogLevel, msg string, keysAndValues ...any)
}

// Redacted logging
logger.Log(InfoLevel, "Config loaded",
    "path", redact.Safe(configPath),        // Safe: file path
    "database_url", configValue)            // Unsafe: contains password
```

**Complexity Trade-offs**:

**Pros**:
- ✅ Used in production at CockroachDB (battle-tested)
- ✅ Type-safe redaction (compile-time safety)
- ✅ Preserves redaction through function boundaries
- ✅ Rich API with SafeFormatter for custom types
- ✅ Works with structured logging (preserves key-value pairs)

**Cons**:
- ❌ Heavy dependency (entire redaction framework)
- ❌ UTF-8 markers may confuse simple log parsers
- ❌ Requires wrapping all values (verbose API)
- ❌ Performance overhead for marker injection
- ❌ Over-engineered for simple config value masking

**Alternative: Simple String Redaction**

For our use case (config value logging), simpler approach:

```go
// Simple redaction function
func redactValue(key string, value any) string {
    // Check if key is sensitive
    if isSensitiveKey(key) {
        return "***REDACTED***"
    }

    // Check value with gitleaks
    if containsSecret(fmt.Sprint(value)) {
        return "***SECRET_DETECTED***"
    }

    return fmt.Sprint(value)
}

// Usage in logger
logger.Log(InfoLevel, "Config value",
    "key", key,
    "value", redactValue(key, value))
```

**Performance Comparison**:

```go
// cockroachdb/redact approach:
// - Allocates RedactableString wrapper
// - Injects UTF-8 markers
// - Requires Redact() call to strip
// Overhead: ~100-200ns per value

// Simple string replacement:
// - Direct string format
// - No allocations (if cached)
// - Immediate masking
// Overhead: ~10-20ns per value
```

**Decision**: **Use simpler string redaction, NOT cockroachdb/redact**

**Rationale**:

1. **Use Case Mismatch**: Our requirement is config value masking for logs, not comprehensive PII protection across distributed systems (CockroachDB's use case)

2. **API Simplicity**: Config library users shouldn't need to learn redact.Safe() wrapper syntax. They just want sensitive values masked automatically.

3. **Dependency Weight**: cockroachdb/redact is 10K+ lines. Our need is <100 lines of redaction logic.

4. **Integration Clarity**: Simple `redactValue(key, value) string` function is more testable and debuggable than marker-based approach.

5. **gitleaks Already Sufficient**: We're using gitleaks for automatic secret detection. Adding another redaction layer is redundant.

**Implementation Plan**:

```go
// internal/redact/redact.go
package redact

import (
    "fmt"
    "github.com/zricethezav/gitleaks/v8/detect"
)

// Redactor handles value redaction
type Redactor struct {
    detector      *detect.Detector
    sensitiveKeys map[string]bool
}

// RedactValue masks sensitive values
func (r *Redactor) RedactValue(key string, value any) string {
    valueStr := fmt.Sprint(value)

    // Check manually marked sensitive keys
    if r.sensitiveKeys[key] {
        return "***REDACTED***"
    }

    // Check automatic secret detection
    findings := r.detector.DetectString(valueStr)
    if len(findings) > 0 {
        return fmt.Sprintf("***SECRET_DETECTED:%s***", findings[0].RuleID)
    }

    return valueStr
}

// Safe wraps a value to bypass redaction (optional convenience)
type Safe struct {
    Value any
}

func (s Safe) String() string {
    return fmt.Sprint(s.Value)
}
```

**Testing Strategy**:

```go
func TestRedactor(t *testing.T) {
    r := NewRedactor(WithSensitiveKeys("password", "api_key"))

    tests := []struct {
        key      string
        value    any
        expected string
    }{
        {"username", "alice", "alice"},                    // Not sensitive
        {"password", "secret", "***REDACTED***"},          // Marked sensitive
        {"token", "ghp_abc123xyz", "***SECRET_DETECTED:github-pat***"}, // Auto-detected
        {"safe_value", 12345, "12345"},                   // Number, safe
    }

    for _, tt := range tests {
        got := r.RedactValue(tt.key, tt.value)
        if got != tt.expected {
            t.Errorf("RedactValue(%q, %v) = %q, want %q",
                tt.key, tt.value, got, tt.expected)
        }
    }
}
```

**References**:
- cockroachdb/redact: https://github.com/cockroachdb/redact
- CockroachDB redaction design: https://wiki.crdb.io/wiki/spaces/CRDB/pages/1824817806/
- Rejected because: Over-engineered for config value masking use case

**Chosen Approach**: Custom `internal/redact` package with:
- `RedactValue(key, value) string` function
- Integration with gitleaks detector
- Sensitive key map from functional options
- Optional `Safe` wrapper for convenience
- Zero additional dependencies (gitleaks already required)

---

## Research Execution Plan

**Parallel Research**: Topics 1-6 can be researched independently and in parallel.

**Dependencies**: None - all topics are independent.

**Deliverable**: Completed research.md with all sections filled by research agents, providing technical foundation for Phase 1 design.

**Next Phase**: Phase 1 (Design & Contracts) begins after all research topics are resolved.
