# Implementation Tasks: Viper Configuration Package

**Feature**: 001-viper-config
**Branch**: `001-viper-config`
**Generated**: 2026-02-14
**Status**: Ready for implementation

## Task Summary

**Total Tasks**: 61
**User Stories**: 4 (US1: Load Config, US2: Mage Integration, US3: Named Args, US4: Consistency)
**Parallel Opportunities**: 25 tasks marked [P]
**Estimated Duration**: 3-5 days (1 developer, sequential) or 2 days (parallel execution)

---

## Phase 1: Setup

**Goal**: Initialize project structure, dependencies, and development tooling.

**Tasks**:

- [x] T001 Create config/ package directory in repository root
- [x] T002 Initialize Go module with `go mod init` if not exists, verify Go 1.23+ in go.mod
- [x] T003 Add dependencies: `go get github.com/spf13/viper@latest github.com/gitleaks/gitleaks/v8 github.com/spf13/pflag@latest`
- [x] T004 [P] Create config/doc.go with package documentation and import path
- [x] T005 [P] Create config/README.md with installation and basic usage examples
- [x] T006 [P] Setup .golangci.yml with linters (gofmt, govet, staticcheck, errcheck)
- [x] T007 [P] Create examples/ directory structure (basic/, with-logger/, mage-integration/)

**Checkpoint**: ✅ Project structure exists, dependencies downloaded, linting configured

---

## Phase 2: Foundational (BLOCKS ALL USER STORIES)

**Goal**: Build core infrastructure required by all user stories - must complete before any story work begins.

**Independent Test**:
- `config.ErrInvalidOption` exists and is an error value
- `Option` type compiles: `var opt config.Option = func(*config) error { return nil }`
- Package imports without errors: `import "github.com/randomvariable/mage-common/config"`

**Tasks**:

- [x] T008 Create config/errors.go with sentinel errors (ErrAlreadyInitialized, ErrInvalidOption, ErrConfigParseFailed)
- [x] T009 Create config/errors_test.go with error identity tests using errors.Is()
- [x] T010 Define config struct in config/config.go with fields: viper, logger, redactor, cleanArgs, initOnce
- [x] T011 Define Option type in config/options.go: `type Option func(*config) error`
- [x] T012 [P] Create config/logger.go with Level type and Logger interface: `Log(ctx context.Context, level Level, msg string, attrs ...any)`
- [x] T013 [P] Create config/logger_test.go with mock logger implementation for testing

**Checkpoint**: ✅ Core types compile, sentinel errors testable, package imports successfully

---

## Phase 3: User Story 1 - Load Standard Configuration

**Priority**: P1
**Story Goal**: Provide Init() function that loads config.yaml gracefully with environment variable overrides

**Independent Test**:
- Create config.yaml, call `config.Init()`, assert `viper.GetString("test.key")` returns expected value
- Set env var `TEST_KEY=envvalue`, call `config.Init()`, assert precedence: env > file
- Call `config.Init()` with missing config.yaml, assert no error returned (graceful)
- Call `config.Init()` twice, assert second call is no-op (idempotent via sync.Once)

**Tasks**:

- [x] T014 [US1] Implement Init() skeleton in config/config.go with sync.Once and package-level singleton
- [x] T015 [US1] Add Viper initialization: create instance, set config name, add default search path "."
- [x] T016 [US1] Implement automatic environment variable binding with viper.AutomaticEnv()
- [x] T017 [US1] Add config file reading with graceful missing file handling (check err != nil, ignore ErrNotExist)
- [x] T018 [US1] Implement Option application loop with early return on validation errors
- [x] T019 [P] [US1] Create config/config_test.go with table-driven tests for Init() acceptance scenarios
- [x] T020 [P] [US1] Implement WithConfigFile() option in config/options.go
- [x] T021 [P] [US1] Implement WithConfigPaths() option with path existence validation
- [x] T022 [P] [US1] Implement WithEnvPrefix() option with viper.SetEnvPrefix()
- [x] T023 [P] [US1] Add tests for WithConfigFile in config/options_test.go
- [x] T024 [P] [US1] Add tests for WithConfigPaths including invalid path handling
- [x] T025 [P] [US1] Add tests for WithEnvPrefix with environment variable precedence

**Checkpoint**: ✅ User Story 1 complete - Init() loads config.yaml, handles env vars, gracefully handles missing file

---

## Phase 4: User Story 2 - Integrate with Mage

**Priority**: P1
**Story Goal**: Prevent Viper/Cobra flags from conflicting with Mage via CleanOSArgs()

**Independent Test**:
- Set `os.Args = []string{"mage", "build", "--image=webapp", "-v"}`, call `CleanOSArgs()`, assert `os.Args = []string{"mage", "build", "-v"}`
- Verify removed flags returned: `[]string{"--image=webapp"}`
- Call with `--flag value` (two args), assert both removed from os.Args
- Verify pflag still captured values before CleanOSArgs() ran

**Tasks**:

- [x] T026 [US2] Implement CleanOSArgs() in config/config.go with os.Args manipulation algorithm
- [x] T027 [US2] Add logic to preserve short flags (-v, -d) while removing long flags (--flag)
- [x] T028 [US2] Handle both `--flag=value` (single arg) and `--flag value` (two args) syntax
- [x] T029 [US2] Return []string of removed flags for debugging/logging
- [x] T030 [P] [US2] Create config/mage_test.go with table-driven tests for CleanOSArgs scenarios
- [x] T031 [P] [US2] Add test for flag preservation: verify short flags remain in os.Args
- [x] T032 [P] [US2] Add test for pflag integration: verify flags captured before cleaning

**Checkpoint**: ✅ User Story 2 complete - CleanOSArgs() prevents Mage conflicts, preserves short flags

---

## Phase 5: User Story 3 - Named Arguments for Mage Targets

**Priority**: P2
**Story Goal**: Enable --key=value syntax for self-documenting Mage target calls

**Independent Test**:
- Define pflag with `pflag.String("image", "", "...")`, parse, clean, init config
- Call `mage build --image=webapp`, verify `viper.GetString("image") == "webapp"`
- Test precedence: CLI flag > env var > config file for same key
- Verify named args work regardless of position: `--image=webapp --tag=latest` == `--tag=latest --image=webapp`

**Tasks**:

- [x] T033 [US3] Document pflag integration pattern in config/doc.go with example code
- [x] T034 [US3] Add pflag binding to Init(): call viper.BindPFlags(pflag.CommandLine) if flags exist
- [x] T035 [US3] Implement precedence handling: ensure CLI flags override env vars and config file
- [x] T036 [P] [US3] Create config/named_args_test.go with pflag integration tests
- [x] T037 [P] [US3] Add test for named argument precedence (flag > env > file)
- [x] T038 [P] [US3] Add test for order independence (verify flag order doesn't matter)

**Checkpoint**: ✅ User Story 3 complete - Named arguments work via pflag with correct precedence

---

## Phase 6: User Story 4 - Cross-Project Consistency

**Priority**: P3
**Story Goal**: Ensure identical behavior across different projects using this library

**Independent Test**:
- Create two separate mock projects (project-a/, project-b/), import config package
- Configure identically, verify same input → same output in both projects
- Import from external module path, verify no hardcoded internal paths
- Verify no init() side effects (explicit Init() required)

**Tasks**:

- [ ] T039 [US4] Add comprehensive GoDoc examples to config/doc.go covering all public functions
- [ ] T040 [US4] Document thread-safety guarantees in config/doc.go (sync.Once, read-only Viper)
- [ ] T041 [US4] Update config/README.md with migration guide from direct Viper usage
- [ ] T042 [P] [US4] Create examples/basic/main.go demonstrating minimal usage (CleanOSArgs + Init)
- [ ] T043 [P] [US4] Create examples/with-logger/main.go with slog adapter integration
- [ ] T044 [P] [US4] Create examples/mage-integration/magefile.go showing full Mage workflow
- [ ] T045 [P] [US4] Create config/consistency_test.go with import safety validation tests
- [ ] T046 [P] [US4] Add multi-project simulation test (create temp modules, verify behavior)

**Checkpoint**: ✅ User Story 4 complete - Documentation comprehensive, examples runnable, consistency validated

---

## Phase 7: Polish & Cross-Cutting Concerns

**Goal**: Add observability features (logging, redaction) and final integration.

**Tasks**:

- [ ] T047 [P] Implement NewSlogAdapter() in config/logger.go with context.Context support
- [ ] T048 [P] Implement NewZerologAdapter() and NewZapAdapter() in config/logger.go
- [ ] T049 Implement WithLogger() option in config/options.go with nil check validation
- [ ] T050 Create config/redaction.go with gitleaks Detector integration
- [ ] T051 Implement WithSensitiveKeys() option: add keys to redactor's manual list
- [ ] T052 [P] Implement WithoutSensitiveKeys() option: remove keys from sensitive list
- [ ] T053 [P] Implement GetSensitiveKeys() in config/config.go returning sorted copy
- [ ] T054 Add logging behavior to Init(): emit "config file loaded", "configuration initialized" with attrs
- [ ] T055 Add logging to CleanOSArgs(): emit "removed long flags" with removed_count and flags list
- [ ] T056 Add debug-level logging: config loading steps, viper binding events, sensitive key detection
- [ ] T057 [P] Create config/redaction_test.go with secret pattern detection tests
- [ ] T058 [P] Add tests for WithSensitiveKeys/WithoutSensitiveKeys/GetSensitiveKeys
- [ ] T059 [P] Create config/logging_test.go verifying all log events match contracts/api.md specification
- [ ] T060 [P] Create config/performance_test.go with benchmarks for Init() and CleanOSArgs() to verify <10ms requirement (SC-006)
- [ ] T061 [P] [US1] Add test in config/config_test.go for nested key access via dot notation (e.g., tools.golangci-lint.version pattern per FR-006)

**Checkpoint**: ✅ All observability features complete - logging, redaction, adapters functional

---

## Dependencies

**User Story Completion Order** (must complete in this sequence):

```
Phase 1 (Setup)
    ↓
Phase 2 (Foundational) ← BLOCKS everything below
    ↓
    ├─→ Phase 3 (US1: Load Config) ← Required by US3
    ├─→ Phase 4 (US2: Mage Integration) ← Independent
    └─→ Phase 7 (Polish) ← Independent
         ↓
Phase 5 (US3: Named Args) ← Requires US1 complete (needs Init + Viper)
    ↓
Phase 6 (US4: Consistency) ← Requires US1, US2, US3 complete (needs all features)
```

**Blocking Tasks**:
- T008-T013 (Phase 2) MUST complete before ANY user story work
- T014-T025 (US1) MUST complete before T033-T038 (US3)
- T014-T025 (US1), T026-T032 (US2), T033-T038 (US3) MUST complete before T039-T046 (US4)

**Non-Blocking** (can run anytime after Phase 2):
- Phase 7 (Polish) can run in parallel with user stories - only needs foundational types

---

## Parallel Execution Examples

**After Phase 2 completes, these can run in parallel**:

### Team A: US1 (Load Config)
```bash
# Work on T014-T025 in parallel
git checkout -b feature/us1-load-config

# Developer 1
- T014-T018 (Init implementation)
- T019 (tests)

# Developer 2
- T020-T022 (options implementation)
- T023-T025 (option tests)
```

### Team B: US2 (Mage Integration)
```bash
# Independent of US1, can work simultaneously
git checkout -b feature/us2-mage-integration

- T026-T029 (CleanOSArgs implementation)
- T030-T032 (tests)
```

### Team C: Polish (Logger + Redaction)
```bash
# Independent of user stories, only needs foundational types
git checkout -b feature/polish-observability

- T047-T048 (adapters)
- T049-T053 (options)
- T057-T059 (tests)
```

**After US1 completes**:

```bash
# Team A continues with US3 (needs Init from US1)
git checkout -b feature/us3-named-args
- T033-T038
```

**After US1, US2, US3 complete**:

```bash
# Final integration and consistency validation
git checkout -b feature/us4-consistency
- T039-T046
```

---

## MVP-First Strategy

**Minimum Viable Product (MVP)**: Complete User Story 1 ONLY

```
MVP = Phase 1 + Phase 2 + Phase 3 (US1)
```

**Why**: US1 provides core value (load config.yaml with env overrides). Test in production before adding complexity.

**Validation Before Expanding**:
1. Deploy MVP to 1-2 pilot projects
2. Gather feedback on Init() API and error handling
3. Validate performance (<10ms, <1MB) in real usage
4. THEN add US2 (Mage integration) and US3 (named args)

**Incremental Delivery Schedule**:
- **Week 1**: Ship US1 (v0.1.0-alpha) → validate core functionality
- **Week 2**: Add US2 (v0.2.0-alpha) → validate Mage compatibility
- **Week 3**: Add US3 (v0.3.0-alpha) → validate named args in real Mage targets
- **Week 4**: Add US4 + Polish (v1.0.0-rc1) → production-ready with observability

---

## Implementation Strategy

### Task Execution Rules

1. **Sequential Phases**: Complete Phase 2 before starting any user story
2. **Parallel Within Stories**: Tasks marked [P] can run simultaneously within same story
3. **Stop at Checkpoints**: Verify "Independent Test" passes before marking story complete
4. **Commit Per Task**: Each task = 1 commit with descriptive message referencing task ID

### Example Workflow

```bash
# Start with foundations
git checkout -b 001-viper-config

# Phase 1: Setup (sequential, quick)
# T001-T007

# Phase 2: Foundational (MUST complete fully)
# T008-T013

# Phase 3: US1 - Load Config (can parallelize T019-T025)
# T014-T025

# Checkpoint: Run independent test
go test -v ./config -run TestInit

# Phase 4: US2 - Mage Integration (parallel with Phase 7)
# T026-T032

# Phase 7: Polish (parallel with Phase 4)
# T047-T059

# Phase 5: US3 - Named Args (needs US1)
# T033-T038

# Phase 6: US4 - Consistency (needs US1+US2+US3)
# T039-T046

# Final validation
go test -v ./config/...
go build ./examples/...
```

### Testing Strategy

- **Unit tests**: One _test.go file per source file, table-driven tests
- **Integration tests**: config/consistency_test.go for multi-component scenarios
- **Example tests**: Verify examples/ compile and run without errors
- **Coverage target**: >80% line coverage for config package

---

## Notes

- **[P] marker**: Task is parallelizable (different files, no blocking dependencies)
- **[US#] label**: Task belongs to User Story # (traceability to spec.md)
- **Independent Test**: Each user story phase has explicit test criteria for completion validation
- **Commit Hygiene**: Reference task IDs in commits: `git commit -m "T014: Implement Init skeleton with sync.Once"`

---

## References

- **Specification**: [spec.md](spec.md) - User stories and acceptance criteria
- **Research**: [research.md](research.md) - Technology decisions (gitleaks, logger interface, functional options)
- **Data Model**: [data-model.md](data-model.md) - Entity definitions and state transitions
- **API Contract**: [contracts/api.md](contracts/api.md) - Complete function signatures and behavior
- **Quickstart**: [quickstart.md](quickstart.md) - User-facing examples and usage patterns
