# Quickstart Guide: Viper Configuration Package

**Feature**: 001-viper-config
**Audience**: Developers using Mage
**Time to Complete**: 5 minutes

## What This Package Does

Provides a standardized way to use Viper for configuration in Mage projects, solving two key problems:

1. **Mage flag conflicts** - Prevents Viper/Cobra flags from breaking Mage target execution
2. **Named arguments** - Enables `mage build --image=webapp` instead of cumbersome positional arguments

## Installation

```bash
go get github.com/randomvariable/mage-common/config
```

## Basic Setup (3 Steps)

### Step 1: Create config.yaml

```yaml
# config.yaml
build:
  platform: linux
  arch: amd64

database:
  host: localhost
  port: 5432
  username: admin
  password: secret123  # Will be auto-redacted in logs
```

### Step 2: Update Your Magefile

```go
//go:build mage

package main

import (
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func init() {
    // A. Define your flags (optional)
    pflag.String("platform", "", "build platform")
    pflag.String("arch", "", "build architecture")
    pflag.Parse()

    // B. Clean os.Args to prevent Mage conflicts
    config.CleanOSArgs()

    // C. Initialize config
    config.Init()
}
```

### Step 3: Use Config in Targets

```go
func Build() error {
    // Read from config file, env vars, or flags
    // Precedence: CLI flags > env vars > config file
    platform := viper.GetString("build.platform")
    arch := viper.GetString("build.arch")

    fmt.Printf("Building for %s/%s\n", platform, arch)
    return nil
}
```

## Usage Examples

### Call with Config File Only

```bash
mage build
# Uses values from config.yaml: linux/amd64
```

### Override with Environment Variables

```bash
BUILD_PLATFORM=darwin mage build
# Uses: darwin/amd64 (platform from env, arch from config)
```

### Override with Command-Line Flags

```bash
mage build --platform=windows --arch=arm64
# Uses: windows/arm64 (both from CLI flags)
```

### Named Arguments Make Targets Self-Documenting

**Before** (positional arguments - confusing):
```bash
mage deploy prod us-east-1 webapp v1.2.3
# What does each argument mean?
```

**After** (named arguments - clear):
```bash
mage deploy --environment=prod --region=us-east-1 --service=webapp --version=v1.2.3
# Self-documenting and order-independent!
```

## Configuration Precedence

Viper uses this order (highest priority first):

1. **Command-line flags**: `mage build --platform=darwin`
2. **Environment variables**: `BUILD_PLATFORM=darwin`
3. **Config file**: `build.platform: darwin` in `config.yaml`
4. **Default values**: From `pflag` definitions

## Optional: Add Logging

```go
import (
    "log/slog"
    "github.com/randomvariable/mage-common/config"
)

func init() {
    logger := config.NewSlogAdapter(slog.Default())

    config.CleanOSArgs()
    config.Init(
        config.WithLogger(logger),
        config.WithSensitiveKeys("database.password", "api.token"),
    )
}
```

**Log Output**:
```
INFO: config loaded from config.yaml
INFO: config_value key=database.host value=localhost
INFO: config_value key=database.password value=***REDACTED***
```

## Redacting Sensitive Values

### Automatic Detection

The package automatically detects and redacts:
- Passwords (any key containing "password", "passwd", "pwd")
- API tokens (keys matching GitHub PAT patterns, JWT tokens, etc.)
- Secrets (keys containing "secret", "key", "credential")
- Other patterns using gitleaks rules

### Manual Marking

Mark additional keys as sensitive:

```go
config.Init(
    config.WithSensitiveKeys(
        "stripe.secret_key",
        "sendgrid.api_key",
        "custom.sensitive_field",
    ),
)
```

**When logged**:
```
config_value key=stripe.secret_key value=***REDACTED***
```

## Environment Variable Mapping

Config keys are automatically mapped to environment variables:

| Config Key | Environment Variable |
|------------|---------------------|
| `build.platform` | `BUILD_PLATFORM` |
| `database.host` | `DATABASE_HOST` |
| `api.retry_count` | `API_RETRY_COUNT` |

**Rules**:
- Dots (`.`) become underscores (`_`)
- All uppercase
- Optional prefix via `WithEnvPrefix("MYAPP")` → `MYAPP_BUILD_PLATFORM`

## Advanced: Custom Config File Location

```go
config.Init(
    config.WithConfigFile("settings"),  // Searches for settings.yaml
    config.WithConfigPaths("/etc/myapp", "$HOME/.config/myapp"),
)
```

**Search Order**:
1. `./settings.yaml`
2. `/etc/myapp/settings.yaml`
3. `$HOME/.config/myapp/settings.yaml`

## Advanced: Multiple Config Formats

Viper supports multiple formats (no code changes needed):

- `config.yaml` (recommended)
- `config.yml`
- `config.json`
- `config.toml`
- `config.env`

Just name your file `config.{extension}` and Viper will detect it.

## Common Patterns

### Pattern 1: Build Config

```yaml
# config.yaml
build:
  image: myapp
  tag: latest
  registry: docker.io
  push: true
```

```go
func Build() error {
    image := viper.GetString("build.image")
    tag := viper.GetString("build.tag")
    registry := viper.GetString("build.registry")

    fullImage := fmt.Sprintf("%s/%s:%s", registry, image, tag)

    // Build image...

    if viper.GetBool("build.push") {
        // Push image...
    }
    return nil
}
```

**Usage**:
```bash
# Use config defaults
mage build

# Override tag
mage build --tag=v1.2.3

# Override everything
mage build --image=webapp --tag=v2.0.0 --registry=gcr.io --push=false
```

---

### Pattern 2: Environment-Specific Config

```yaml
# config.yaml
default: &default
  log_level: info
  debug: false

dev:
  <<: *default
  log_level: debug
  debug: true
  database:
    host: localhost

prod:
  <<: *default
  database:
    host: prod-db.internal
    port: 5432
```

```go
func Deploy() error {
    env := viper.GetString("environment") // From --environment flag

    // Load environment-specific config
    dbHost := viper.GetString(env + ".database.host")
    debugMode := viper.GetBool(env + ".debug")

    // Deploy with environment settings...
}
```

**Usage**:
```bash
mage deploy --environment=dev
mage deploy --environment=prod
```

---

### Pattern 3: Tool Versions

```yaml
# config.yaml
tools:
  go: 1.23.1
  golangci-lint: 1.55.2
  buf: 1.28.1
  kubectl: 1.28.0
```

```go
func InstallTools() error {
    goVersion := viper.GetString("tools.go")
    lintVersion := viper.GetString("tools.golangci-lint")

    // Install specific versions...
}
```

---

### Pattern 4: Feature Flags

```yaml
# config.yaml
features:
  new_ui: true
  experimental_api: false
  beta_metrics: true
```

```go
func Test() error {
    if viper.GetBool("features.experimental_api") {
        // Run experimental API tests
    }

    if viper.GetBool("features.beta_metrics") {
        // Run metrics tests
    }
}
```

**Usage**:
```bash
# Enable experimental features for testing
mage test --features.experimental_api=true
```

---

## Troubleshooting

### Issue: "flag provided but not defined"

**Cause**: Forgot to call `config.CleanOSArgs()`

**Fix**:
```go
func init() {
    pflag.Parse()
    config.CleanOSArgs() // ← Add this
    config.Init()
}
```

---

### Issue: Environment variables not working

**Cause**: Key name doesn't match expected format

**Solution**: Check environment variable naming:
- Config: `database.host` → Env: `DATABASE_HOST`
- Config: `api.retry-count` → Env: `API_RETRY_COUNT` (dash becomes underscore)

**Debugging**:
```go
// Print all environment variables Viper is using
for _, key := range viper.AllKeys() {
    fmt.Printf("%s = %v (source: %s)\n",
        key,
        viper.Get(key),
        viper.GetViper().ConfigFileUsed(), // Shows which source
    )
}
```

---

### Issue: Config file not found (but should be there)

**Cause**: Viper is searching in wrong directory

**Fix**: Add explicit path
```go
config.Init(
    config.WithConfigPaths("."), // Current directory
)
```

**Debug**: Print search paths
```go
fmt.Println("Config paths:", viper.ConfigFileUsed())
```

---

### Issue: Values not overriding correctly

**Check precedence**:
```go
fmt.Println("From config file:", viper.GetString("build.platform"))
fmt.Println("From env:", os.Getenv("BUILD_PLATFORM"))
fmt.Println("From flag:", pflag.Lookup("platform").Value.String())
fmt.Println("Final value:", viper.GetString("build.platform"))
```

Remember: **Flags > Env Vars > Config File > Defaults**

---

## Migration Guide

### Migrating from Environment Variables Only

**Before**:
```go
func Build() {
    platform := os.Getenv("BUILD_PLATFORM")
    if platform == "" {
        platform = "linux" // Manual default
    }
}
```

**After**:
```go
// In init()
config.Init()

// In target
func Build() {
    platform := viper.GetString("build.platform") // Auto-default from config
}
```

**Benefits**:
- ✅ Centralized config in `config.yaml`
- ✅ No manual default handling
- ✅ CLI flag support for free

---

### Migrating from Positional Arguments

**Before**:
```go
func Deploy(ctx context.Context, environment, region string) error {
    // Called as: mage deploy prod us-east-1
}
```

**After**:
```go
func init() {
    pflag.String("environment", "dev", "deployment environment")
    pflag.String("region", "us-east-1", "AWS region")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()
}

func Deploy() error {
    env := viper.GetString("environment")
    region := viper.GetString("region")

    // Called as: mage deploy --environment=prod --region=us-west-2
}
```

**Benefits**:
- ✅ Self-documenting (`--environment` instead of positional mystery)
- ✅ Order-independent
- ✅ Config file defaults
- ✅ Environment variable support

---

## Best Practices

### ✅ DO:
- Call `CleanOSArgs()` in `init()` before `config.Init()`
- Define flags with `pflag` before parsing
- Use `config.yaml` for defaults, env vars for deployment, flags for overrides
- Mark sensitive keys explicitly with `WithSensitiveKeys()`
- Use structured config (nested keys) for organization

### ❌ DON'T:
- Don't call `config.Init()` multiple times (it's idempotent but wasteful)
- Don't mix `flag` and `pflag` packages (use pflag only)
- Don't hardcode sensitive values in `config.yaml` (use env vars or secret management)
- Don't commit `.env` files or `config.local.yaml` to version control

---

## Next Steps

- See [API documentation](contracts/api.md) for complete API reference
- See [Data Model](data-model.md) for internal architecture
- See [Observability Checklist](checklists/observability.md) for logging best practices

---

## Need Help?

Common questions:

**Q: Can I use multiple config files?**
A: Yes, Viper will merge them. Use `WithConfigPaths()` to add search paths.

**Q: How do I reset config between tests?**
A: Use `viper.Reset()` in test setup (not recommended in production code).

**Q: Can I watch for config file changes?**
A: Yes, use `viper.WatchConfig()` - but not recommended for Mage (short-lived CLI app).

**Q: What if I need different config per Mage target?**
A: Use nested config structure and read target-specific keys:
```yaml
targets:
  build:
    timeout: 5m
  test:
    timeout: 10m
```

```go
func Build() {
    timeout := viper.GetDuration("targets.build.timeout")
}
```

---

**Last Updated**: 2026-02-14
**Version**: 1.0.0-dev
