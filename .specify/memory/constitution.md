<!--
Sync Impact Report:
- Version: 1.3.0 → 1.4.0
- Version Bump Rationale: MINOR - Added Principle X for named arguments in Mage targets
- Modified Principles:
  * X. Named Arguments for Mage Targets → New principle requiring config-driven named arguments for all Mage target parameters
- Templates Status:
  ✅ plan-template.md: no impact
  ✅ spec-template.md: no impact
  ✅ tasks-template.md: no impact
- Follow-up: None - all changes synchronized
-->

# mage-common Constitution

## Core Principles

### I. Library-First Architecture

Every feature MUST be implemented as an importable Go library package. Each package:

- MUST be self-contained with clear boundaries
- MUST have a single, well-defined purpose
- MUST NOT depend on project-specific implementation details
- MUST be independently testable without requiring a complete project setup
- MUST be reusable across multiple projects

**Rationale**: This library exists to extract common Mage and tooling patterns for reuse. Utilities that only work in one context defeat this purpose.

### II. Viper-Based Configuration

Configuration loading MUST use Viper with standardized patterns:

- Config file location: `config.yaml` in project root
- Config reading MUST be optional (gracefully handle missing config)
- Mage integration MUST use `CleanOSArgs()` to prevent flag conflicts
- Named arguments for Mage targets MUST be supported via Viper
- Environment variable overrides MUST follow standard patterns

**Rationale**: Viper provides consistent configuration across projects while enabling named arguments for Mage targets, solving the positional argument limitation.

### III. Declarative Tool Management

Tool installation and versioning MUST be declarative and reproducible:

- Tools MUST be defined in a `.tools.yaml` config with: `name`, `version`, and one or more `sources` (priority-ordered)
- Each source specifies a `type` and type-specific fields (URL, package name, local path, etc.)
- Supported install types: `go` (go install), `gem` (Bundler binstubs), `npx` (pinned npx), `cargo` (cargo install), `uvx` (uv package manager), `download` (HTTP with checksum verification)
- Tools MUST be installed to a configurable tools directory (default `hack/bin/{GOOS}/{GOARCH}/`) with install-type-appropriate versioning (symlinks for go/cargo, binstubs for gem, shims for npx/uvx)
- `InstallAll()` MUST be idempotent (skip already-installed versions)
- Tool path prepending MUST be available via `PrependToPath()`

**Rationale**: Version-pinned tools prevent "works on my machine" issues and enable reproducible builds across environments.

### IV. Production-Grade Error Handling

All exported functions MUST follow Go error handling best practices:

- Return `error` as the last return value
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Define package-level static sentinel errors for classification (e.g., `var ErrUnknownInstallType = errors.New("unknown install type")`)
- MUST use static sentinel errors with wrapping for error context
- MUST NOT use dynamic error creation with `errors.New()` in library code (define sentinels at package level)
- MUST NOT use `panic()` except for programmer errors during init
- MUST NOT silently ignore errors without explicit justification
- All errors MUST be checked and handled appropriately, even in test code

**Rationale**: Production code requires reliable error propagation and clear failure modes for debugging. Static sentinel errors enable proper error type checking with `errors.Is()` and `errors.As()` for callers.

### V. Import Compatibility

All packages MUST be safely importable by external projects:

- MUST NOT use `init()` functions with side effects
- MUST NOT perform I/O, network calls, or system modifications during import
- MUST NOT require specific working directories or file structures
- Configuration initialization MUST be explicit (e.g., `InitConfig()` called by user)
- MUST support multiple module paths (github.com, internal enterprise repos)

**Rationale**: Init-time side effects break when imported by projects with different structures. Explicit initialization gives callers control.

### VI. Test-Driven Development (NON-NEGOTIABLE)

Testing is mandatory for all exported functionality:

- Unit tests MUST exist for all exported functions
- Test coverage MUST be ≥80% for core logic paths
- Tests MUST use table-driven patterns where applicable
- All tests MUST call `t.Parallel()` for safe concurrent execution
- Test helpers MUST call `t.Helper()` to provide accurate line numbers
- Tests MUST use clear, descriptive test case names
- Tests MUST NOT require external dependencies (network, filesystem) unless integration tests
- Integration tests MUST be separated with build tags (e.g., `//go:build integration`)
- Mock interfaces MUST be provided for external dependencies
- All error returns MUST be checked, even in test code

**Rationale**: This is a library used by multiple projects. Untested code creates downstream breakage. Parallel tests catch race conditions and ensure thread-safety.

### VII. Documentation Standards

All public APIs MUST be documented following godoc conventions:

- Every exported type, function, constant, and variable MUST have a GoDoc comment
- GoDoc comments MUST start with the identifier name (e.g., `// PackageName does...`)
- Package documentation MUST start with `// Package <name> ...` and include usage examples
- Package documentation MUST include: installation instructions, quick start guide, common patterns
- Breaking changes MUST be documented in CHANGELOG.md with migration guides
- Include examples for common usage patterns using Go's `Example` test functions

**Rationale**: Library users need self-service documentation. Poor docs create support burden. Following godoc conventions ensures documentation appears correctly in pkg.go.dev.

### VIII. Code Quality Standards

Code quality is enforced through linting and consistent practices:

**Linting Policy (NON-NEGOTIABLE)**:
- MUST NEVER add exclusions to golangci-lint configuration (`.golangci.yml`)
- When golangci-lint reports an issue: fix the code, refactor to eliminate the problem, or improve code quality to meet standards
- MUST NOT add exclusion rules to disable specific linter checks for convenience
- MUST NOT use `nolint` directives except in documented exceptional cases
- If a linter rule is consistently problematic, it MUST be disabled globally with documented rationale, not worked around with exclusions

**Code Organization**:
- Prefer explicit error handling over inline error checks
- Use modern Go idioms (e.g., `any` instead of `interface{}`)
- Avoid overly complex nested conditionals (maximum cyclomatic complexity: 15, gocyclo default)
- Use constants for repeated string literals (detected by goconst)
- Maximum line length: 180 characters (golines)
- Avoid deep nesting: prefer early returns and guard clauses

**Rationale**: Exclusions hide problems rather than solving them. They create technical debt and allow code quality to degrade over time. Consistent code patterns improve maintainability and reduce cognitive load.

## Go Standards

### Code Quality

- golangci-lint MUST pass with the project's `.golangci.yml` configuration
- Code MUST follow standard Go formatting (gofmt, gofumpt, gci, goimports)
- License headers (Apache 2.0) MUST be present on all `.go` files
- SPDX identifier MUST be included in license headers
- Maximum line length: 180 characters (golines)

### Module Organization

- Module path: `github.com/randomvariable/mage-common`
- Package structure: flat, domain-organized (e.g., `config/`, `tools/`, `output/`)
- Internal packages: use `internal/` for non-exported shared code
- Avoid deep nesting: prefer flat structure over hierarchical

## Development Workflow

### Feature Development

1. Create feature spec using `/speckit.specify`
2. Write implementation plan using `/speckit.plan`
3. Generate tasks using `/speckit.tasks`
4. **Write tests first** (per Principle VI)
5. Implement to pass tests
6. Run `mage lint` and fix issues
7. Update documentation
8. Submit PR with test evidence

### Code Review Requirements

All PRs MUST:

- Pass `mage lint` (golangci-lint + header checks)
- Pass all unit tests (`go test ./...`)
- Include test coverage report for new code
- Update relevant documentation (README, GoDoc, CHANGELOG)
- Have approval from at least one maintainer

## Governance

This constitution supersedes all other development practices. Changes to this constitution require:

1. Documented rationale for the change
2. Impact analysis on existing code and dependent projects
3. Migration plan if breaking changes required
4. Approval from project maintainer

All feature work and code reviews MUST verify compliance with these principles. Exceptions require explicit justification in PR descriptions.

### IX. Verbose Build Output

Mage targets and tool execution MUST produce verbose, copy-pasteable output:

- The command runner MUST print the effective working directory
- The command runner MUST print the full absolute path to the executable
- The command runner MUST print all arguments
- The command runner MUST print only **added** environment variables (not inherited process env)
- Secret values in arguments and environment variables MUST be redacted
- Output MUST be formatted as a single copy-pasteable shell command (e.g., `cd /path && VAR=val /abs/path/to/binary arg1 arg2`)

**Rationale**: Verbose output enables fast debugging by allowing developers to copy-paste the exact command into a terminal to reproduce issues. Showing only added env vars avoids noise from the inherited process environment.

**Complexity Guideline**: Start simple. Prefer straightforward implementations over premature abstraction. Add complexity only when patterns emerge across multiple use cases.

**Runtime Guidance**: For detailed development practices, see CLAUDE.md in the project root.

### X. Named Arguments for Mage Targets

All Mage targets that accept parameters MUST use named arguments via the config package:

- Parameters MUST be read from Viper (`viper.GetString()`, `viper.GetBool()`, etc.) rather than positional Mage arguments
- Flag registration MUST use `pflag.String()`/`pflag.Bool()`/etc. on `pflag.CommandLine`
- Imported target packages MUST register their flags in `init()` (flag registration is a safe init-time operation — no I/O, consistent with Principle V)
- Consuming magefiles MUST call `pflag.Parse()` followed by `config.CleanOSArgs()` in their own `init()` to strip long flags before Mage processes arguments
- Target functions MUST call `config.Init()` before reading Viper values (idempotent via `sync.Once`)
- Sensible defaults MUST be provided at flag registration time
- Required parameters with no sensible default MUST validate non-empty and return a clear usage error
- Flag names MUST be domain-specific to avoid environment variable collisions (e.g., `--tool` not `--name`, since Viper's `AutomaticEnv()` maps generic names like `name` to `NAME` which is commonly set in shells)

**Precedence** (highest to lowest):
1. CLI flags (`--tool=foo`)
2. Environment variables (`TOOL=foo`)
3. Configuration file (`config.yaml`)
4. Default values (from pflag registration)

**Initialization Sequence**:
```
init()  → pflag.String("tool", "", "...")      // register flags (domain-specific names)
init()  → pflag.Parse()                        // parse os.Args
init()  → config.CleanOSArgs()                 // strip --flags from os.Args for Mage
target  → config.Init()                        // bind pflags to Viper (lazy, idempotent)
target  → viper.GetString("tool")              // read value
```

**Consumer Pattern**:
```go
//go:build mage

package main

import (
    "github.com/spf13/pflag"
    "github.com/randomvariable/mage-common/config"
    //mage:import tools
    _ "github.com/randomvariable/mage-common/tools/targets"
)

func init() {
    pflag.Parse()
    config.CleanOSArgs()
}
```

**Rationale**: Named arguments provide a superior UX over positional arguments by supporting config file defaults, environment variable overrides, and self-documenting `--flag` syntax. The pflag → CleanOSArgs → Init → Viper chain integrates cleanly with Mage's argument processing while maintaining the precedence hierarchy defined in Principle II.

---

**Version**: 1.4.0 | **Ratified**: 2026-02-14 | **Last Amended**: 2026-02-14
