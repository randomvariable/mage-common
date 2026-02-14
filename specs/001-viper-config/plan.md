# Implementation Plan: Viper Configuration Package

**Branch**: `001-viper-config` | **Date**: 2026-02-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-viper-config/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement a reusable Go library package that standardizes Viper configuration loading across Mage-based projects. The package solves flag parsing conflicts between Mage and Viper, enables named arguments for Mage targets, and provides consistent config patterns (config.yaml + environment overrides). Core capability: `Init()` loads config.yaml gracefully, `CleanOSArgs()` prevents Mage conflicts, optional logger interface enables visibility, and proven secret detection (gitleaks) protects sensitive values in logs.

## Technical Context

**Language/Version**: Go 1.23+ (per constitution requirement)
**Primary Dependencies**:
- `github.com/spf13/viper` - Configuration management with YAML/env support
- `github.com/gitleaks/gitleaks/v8/detect` - Secret detection (chosen in research phase)
- `github.com/cockroachdb/redact` - Value redaction patterns (implementation reference only, NOT used as dependency)

**Storage**: File-based (config.yaml), environment variables
**Testing**: Go standard testing package with table-driven tests
**Target Platform**: Cross-platform (Linux, macOS, Windows) via Go's cross-compilation
**Project Type**: Single library package
**Performance Goals**: <10ms configuration loading time for typical files (<100 keys)
**Constraints**: <1MB memory footprint for loaded config data, zero performance impact after initialization
**Scale/Scope**: Support projects with <100 config keys, 10-20 environment variable overrides, 5-10 sensitive keys marked

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Evidence / Notes |
|-----------|--------|------------------|
| **I. Library-First Architecture** | ✅ PASS | Package is self-contained, independent library with single purpose (Viper config loading). No project-specific dependencies. Spec §FR-007 requires importability by external projects. |
| **II. Viper-Based Configuration** | ✅ PASS | This IS the Viper configuration package. Implements standardized patterns: config.yaml in root, optional config (FR-002), environment overrides (FR-003), CleanOSArgs (FR-004). |
| **III. Declarative Tool Management** | N/A | Feature is configuration loading, not tool management. No tools installed by this package. |
| **IV. Production-Grade Error Handling** | ✅ PASS | Spec Clarifications (Q5): Init() returns error for all failures, wraps errors with context (FR-009), never panics, defines error values for classification. No silent ignores. |
| **V. Import Compatibility** | ✅ PASS | Spec §FR-007 explicitly requires no init() side effects, explicit initialization via Init() call, no file structure assumptions, no hardcoded paths (FR-007, FR-035, FR-036 in api checklist). |
| **VI. Test-Driven Development** | ✅ PASS | Spec provides detailed acceptance scenarios (User Stories 1-4), independent tests, edge cases. Success criteria are measurable (SC-001 through SC-006). Table-driven tests planned for multiple scenarios. |
| **VII. Documentation Standards** | ✅ PASS | Spec §Usability requires GoDoc comments (FR-052 in api checklist), usage examples, error messages with actionable guidance. README will include installation, quick start, common patterns. |

**Gate Result**: ✅ **PASS** - All applicable principles satisfied. No violations requiring justification.

**Post-Phase 1 Re-Check**: Will verify error handling implementation matches pattern, logger interface design is idiomatic, functional options pattern is correctly applied.

## Project Structure

### Documentation (this feature)

```text
specs/001-viper-config/
├── spec.md              # Feature specification (complete)
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (to be generated)
├── data-model.md        # Phase 1 output (to be generated)
├── quickstart.md        # Phase 1 output (to be generated)
├── contracts/           # Phase 1 output (API contracts, to be generated)
├── checklists/          # Quality gates (existing)
│   ├── requirements.md  # Spec quality checklist
│   ├── api.md           # API requirements checklist
│   └── observability.md # Observability requirements checklist
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Go library package structure (flat, domain-organized per constitution)

config/                  # Main package directory
├── config.go            # Core Init(), CleanOSArgs() implementation
├── config_test.go       # Unit tests (table-driven)
├── options.go           # Functional options (WithSensitiveKeys, WithLogger)
├── options_test.go      # Options pattern tests
├── logger.go            # Logger interface definition
├── redaction.go         # Secret detection & redaction logic
├── redaction_test.go    # Redaction tests with secret patterns
├── errors.go            # Package-level error variables
├── doc.go               # Package documentation + examples
└── README.md            # Package README (usage examples)

internal/                # Non-exported shared code (if needed)
└── testing/             # Test helpers (optional)
    └── fixtures.go      # Reusable test fixtures

examples/                # Usage examples
├── basic/
│   └── main.go          # Basic Init() example
├── with-logger/
│   └── main.go          # Logging integration example
└── mage-integration/
    └── magefile.go      # Mage target with CleanOSArgs example
```

**Structure Decision**: Flat single-package layout per constitution Module Organization. The `config/` package contains all functionality. No deep nesting. Internal package only if private helpers emerge during implementation. Examples directory demonstrates real-world usage patterns for different scenarios (basic, logging, Mage integration).

## Complexity Tracking

**No violations** - Constitution Check passed all applicable principles. No complexity justification required.

---

## Phase 0: Research (COMPLETED)

All research topics resolved. See [research.md](research.md) for details.

**Completed Topics**:
1. ✅ **Secret Detection Library**: Chose gitleaks over trufflehog (MIT license, better API, performance)
2. ✅ **Functional Options Pattern**: WithX naming, error returns, unexported config struct
3. ✅ **Logger Interface Design**: Single Log method with adapters for slog/zerolog/zap
4. ✅ **Viper Integration Patterns**: Precedence order, environment variables, thread-safety from  reference
5. ✅ **Mage Flag Handling**: CleanOSArgs implementation from code
6. ✅ **Redaction Library**: Rejected cockroachdb/redact as over-engineered, implementing simple string redaction

---

## Phase 1: Design & Contracts (COMPLETED)

All design artifacts generated. See individual files for details.

**Generated Artifacts**:
- ✅ **data-model.md**: Core entities (Config, Option, Logger, Redactor), state transitions, validation rules
- ✅ **contracts/api.md**: Complete public API surface with signatures, examples, thread-safety guarantees
- ✅ **quickstart.md**: User-facing guide with 3-step setup, usage examples, migration patterns

**Agent Context**: Updated GitHub Copilot instructions with new technologies from plan.

---

## Post-Design Constitution Re-Check

*Re-evaluating constitution compliance after Phase 1 design completion.*

| Principle | Status | Design Verification |
|-----------|--------|---------------------|
| **I. Library-First Architecture** | ✅ PASS | **API Contract**: `Init(opts ...Option) error` with no hardcoded paths. **Data Model**: Package-level singleton, zero external dependencies beyond Viper/gitleaks. Public API via `contracts/api.md` is self-contained. |
| **II. Viper-Based Configuration** | ✅ PASS | **API Contract**: Wraps Viper global instance, delegates all Get* calls to Viper. **Data Model**: config struct owns viper.Viper instance, initialization follows standard Viper patterns (AddConfigPath, SetConfigName, BindPFlags, ReadInConfig). |
| **III. Declarative Tool Management** | N/A | No tools managed. |
| **IV. Production-Grade Error Handling** | ✅ PASS | **API Contract**: All Option functions return error. Init() wraps errors with context (ErrInvalidOption, ErrConfigParseFailed). **Data Model**: Error propagation documented, no silent failures, no panics in contract guarantees. |
| **V. Import Compatibility** | ✅ PASS | **API Contract**: No init() side effects, explicit Init() call required. **Data Model**: sync.Once ensures Init() idempotency, thread-safe. No assumptions about caller's project structure (WithConfigPaths allows custom paths). |
| **VI. Test-Driven Development** | ✅ PASS | **Data Model**: Validation rules table-driven (logger nil check, sensitive keys validation). **API Contract**: Examples include test assertions. Acceptance criteria from spec map directly to test cases. |
| **VII. Documentation Standards** | ✅ PASS | **Quickstart**: 3-step setup, 11 usage examples, troubleshooting section. **API Contract**: Every function documented with purpose, parameters, returns, examples. **Data Model**: Performance benchmarks specified, thread-safety guarantees explicit. |

**Design Validation**: ✅ **PASS** - All constitution principles satisfied in design phase.

**Key Design Decisions Validated**:
1. **Error Handling**: Matches constitution pattern (wrapped errors, no panics) - §API Contract errors section
2. **Logger Interface**: Idiomatic Go (single method, variadic attrs) - §Data Model Logger entity
3. **Functional Options**: Correctly applied (Option func returning error) - §API Contract Option type
4. **Thread Safety**: sync.Once for Init(), read-only Viper access - §Data Model State Transitions
5. **Dependency Management**: Only stdlib + Viper + gitleaks (no transitive bloat) - §Technical Context

**No issues** - Design is ready for Phase 2 (task generation) and implementation.

---

## Next Steps

**Phase 2: Task Generation** (not executed by `/speckit.plan`)

Run `/speckit.tasks` to generate dependency-ordered implementation tasks based on:
- API contracts from `contracts/api.md`
- Data model from `data-model.md`
- Research decisions from `research.md`
- Acceptance criteria from `spec.md`

**Implementation Readiness**: All unknowns resolved, all design artifacts complete, constitution validated.
