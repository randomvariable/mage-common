# config

Standardized Viper-based configuration loading for Mage projects.

## Features

- **Graceful Config Loading**: Load `config.yaml` with automatic fallback if missing
- **Environment Override**: Environment variables automatically override file values
- **Mage Integration**: `CleanOSArgs()` prevents Viper/Cobra flags from conflicting with Mage
- **Named Arguments**: Enable `--key=value` syntax for self-documenting Mage targets
- **Structured Logging**: Optional logger integration with sensitive value redaction
- **Import Safe**: No `init()` side effects, works in any project structure

## Installation

```bash
go get github.com/randomvariable/mage-common/config
```

## Quick Start

### Basic Configuration Loading

```go
package main

import (
    "fmt"
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/viper"
)

func main() {
    // Initialize configuration (loads config.yaml + environment variables)
    if err := config.Init(); err != nil {
        panic(fmt.Sprintf("config init failed: %v", err))
    }

    // Access configuration values using Viper
    version := viper.GetString("tools.golangci-lint.version")
    fmt.Printf("golangci-lint version: %s\n", version)
}
```

### Mage Target with Named Arguments

```go
//go:build mage

package main

import (
    "fmt"
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

// Build builds the Docker image
// Usage: mage build --image=webapp --tag=v1.2.3
func Build() error {
    // Define command-line flags
    pflag.String("image", "", "Image name to build")
    pflag.String("tag", "latest", "Image tag")
    pflag.Parse()

    // Remove long flags from os.Args to prevent Mage conflicts
    removed := config.CleanOSArgs()

    // Initialize config (binds pflag values to Viper)
    if err := config.Init(); err != nil {
        return err
    }

    // Access named arguments via Viper (CLI > env > config file)
    imageName := viper.GetString("image")
    tag := viper.GetString("tag")

    fmt.Printf("Building %s:%s\n", imageName, tag)
    // ... build logic here
    return nil
}
```

### With Logger and Sensitive Keys

```go
import (
    "log/slog"
    "os"
    "github.com/randomvariable/mage-common/config"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    err := config.Init(
        config.WithLogger(config.NewSlogAdapter(logger)),
        config.WithSensitiveKeys("api.token", "db.password"),
        config.WithEnvPrefix("APP"),
    )
    if err != nil {
        panic(err)
    }
}
```

## Global vs Target-Specific Arguments

### Global Arguments (All Targets)

Define flags in `init()` to make them available to all Mage targets:

```go
//go:build mage

package main

import (
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func init() {
    // Global flags available to ALL targets
    pflag.Bool("verbose", false, "Enable verbose logging")
    pflag.String("environment", "dev", "Deployment environment (dev/staging/prod)")
    pflag.String("config-file", "config", "Configuration file name")

    // Parse flags and initialize config
    pflag.Parse()
    config.CleanOSArgs()

    if err := config.Init(config.WithConfigFile(viper.GetString("config-file"))); err != nil {
        panic(err)
    }
}

// All targets can access global flags
// Usage: mage build --verbose --environment=prod
func Build() error {
    if viper.GetBool("verbose") {
        fmt.Println("Verbose mode enabled")
    }
    env := viper.GetString("environment")
    fmt.Printf("Building for %s\n", env)
    return nil
}

// Usage: mage test --verbose --environment=staging
func Test() error {
    if viper.GetBool("verbose") {
        fmt.Println("Running tests in verbose mode")
    }
    env := viper.GetString("environment")
    fmt.Printf("Testing %s environment\n", env)
    return nil
}
```

### Target-Specific Arguments

Define flags inside the target function for arguments only that target needs:

```go
//go:build mage

package main

import (
    "github.com/randomvariable/mage-common/config"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
)

func init() {
    // Global flags (if any)
    pflag.Bool("verbose", false, "Enable verbose logging")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()
}

// Build has target-specific flags
// Usage: mage build --image=webapp --tag=v1.0 --push
func Build() error {
    // Define flags specific to this target
    pflag.String("image", "", "Container image name (required)")
    pflag.String("tag", "latest", "Image tag")
    pflag.Bool("push", false, "Push image to registry after build")

    // Re-parse to capture target-specific flags
    pflag.Parse()
    config.CleanOSArgs()

    // Re-initialize to bind new flags
    // Note: Init() is idempotent, subsequent calls update flag bindings
    if err := config.Init(); err != nil {
        return err
    }

    image := viper.GetString("image")
    if image == "" {
        return fmt.Errorf("--image is required")
    }

    tag := viper.GetString("tag")
    push := viper.GetBool("push")

    fmt.Printf("Building %s:%s (push=%v)\n", image, tag, push)
    return nil
}

// Deploy has different target-specific flags
// Usage: mage deploy --cluster=prod-us-west --namespace=default
func Deploy() error {
    pflag.String("cluster", "", "Kubernetes cluster (required)")
    pflag.String("namespace", "default", "Kubernetes namespace")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()

    cluster := viper.GetString("cluster")
    if cluster == "" {
        return fmt.Errorf("--cluster is required")
    }

    ns := viper.GetString("namespace")
    fmt.Printf("Deploying to %s/%s\n", cluster, ns)
    return nil
}
```

### When to Use Each Pattern

**Use Global Arguments When:**
- Flag applies to all or most targets (e.g., `--verbose`, `--dry-run`, `--environment`)
- Need consistent behavior across entire build (e.g., `--config-file`, `--log-level`)
- Setting affects infrastructure/tooling shared by all targets

**Use Target-Specific Arguments When:**
- Flag only makes sense for one target (e.g., `--image` for Build, `--cluster` for Deploy)
- Want to prevent flag pollution (avoid showing irrelevant flags in help)
- Different targets need same flag name with different meanings
- Target has many specialized options

**Best Practice Example:**

```go
func init() {
    // Global: affects all targets
    pflag.Bool("verbose", false, "Verbose output")
    pflag.String("env", "dev", "Environment")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()
}

func Build() error {
    // Target-specific: only Build needs these
    pflag.String("image", "", "Image name")
    pflag.String("tag", "latest", "Image tag")
    pflag.Parse()
    config.CleanOSArgs()
    config.Init()

    // Access both global and target-specific flags
    verbose := viper.GetBool("verbose")  // global
    image := viper.GetString("image")     // target-specific
    // ...
}
```

## Configuration Precedence

Values are loaded in the following order (highest to lowest priority):

1. **Command-line flags** (via pflag binding)
2. **Environment variables** (automatic with optional prefix)
3. **Configuration file** (`config.yaml` by default)
4. **Default values**

## Configuration File Format

Example `config.yaml`:

```yaml
tools:
  golangci-lint:
    version: "1.62.2"
    enabled: true

api:
  endpoint: "https://api.example.com"
  timeout: 30s

db:
  host: "localhost"
  port: 5432
```

Access via dot notation: `viper.GetString("tools.golangci-lint.version")`

## Environment Variables

Environment variables automatically override config file values:

```bash
# Override tools.golangci-lint.version
export TOOLS_GOLANGCI_LINT_VERSION="1.63.0"

# With custom prefix (using WithEnvPrefix("APP"))
export APP_API_ENDPOINT="https://prod.api.example.com"
```

## Options

### Available Options

- `WithConfigFile(name string)` - Override default config file name (default: "config")
- `WithConfigPaths(paths ...string)` - Add additional search paths for config files
- `WithEnvPrefix(prefix string)` - Set environment variable prefix
- `WithLogger(logger config.Logger)` - Enable structured logging
- `WithSensitiveKeys(keys ...string)` - Mark config keys as sensitive for redaction
- `WithoutSensitiveKeys(keys ...string)` - Unmark previously sensitive keys

### Logger Adapters

Built-in adapters for popular logging libraries:

- `NewSlogAdapter(logger *slog.Logger)` - Standard library `log/slog`
- `NewZerologAdapter(logger *zerolog.Logger)` - `github.com/rs/zerolog`
- `NewZapAdapter(logger *zap.Logger)` - `go.uber.org/zap`

## Thread Safety

- `Init()` uses `sync.Once` internally - safe to call multiple times, executes once
- After `Init()` completes, all Viper reads are safe for concurrent access
- **Do not call `viper.Set()` after Init()** - configuration is read-only after initialization

## Error Handling

Init() returns:
- `config.ErrAlreadyInitialized` - Config already initialized (harmless, can ignore)
- `config.ErrInvalidOption` - Option validation failed (check option arguments)
- `config.ErrConfigParseFailed` - Config file parse error (check YAML syntax)

```go
if err := config.Init(); err != nil {
    if errors.Is(err, config.ErrAlreadyInitialized) {
        // Harmless - already initialized
        return nil
    }
    return fmt.Errorf("config initialization failed: %w", err)
}
```

## Examples

See [examples/](../examples/) directory for complete working examples:

- `basic/` - Simple config loading
- `with-logger/` - Integrated logging
- `mage-integration/` - Full Mage target example

## License

Same license as parent project.
