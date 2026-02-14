# Tasks: YAML-Configured Tool Dependency Installation

**Input**: Design documents from `/specs/002-tool-install/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/tools.go, quickstart.md

**Tests**: Included per constitution (Principle VI: TDD). Table-driven, `t.Parallel()`, integration tests with build tags.

**Organization**: Tasks grouped by user story in priority order. US1+US2 combined (US2 is a subset of US1). US5 (multi-source fallback) integrated into US1 since fallback is core installer logic.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US8)
- Exact file paths included in descriptions

---

## Phase 1: Setup

**Purpose**: Add dependencies, create directory structure, configure tooling

- [X] T001 Add `k8s.io/apimachinery` v0.35.x dependency in go.mod (`go get k8s.io/apimachinery@v0.35.0`)
- [X] T002 Promote `hashicorp/go-version` from indirect to direct dependency in go.mod (`go get github.com/hashicorp/go-version@v1.8.0`)
- [X] T003 Create directory structure: `api/tools/v1alpha1/scheme/`, `tools/`, `internal/testutil/`
- [X] T004 [P] Create kube-api-linter config in `.custom-gcl.yml` per research R-007 (module plugin with `sigs.k8s.io/kube-api-linter`)
- [X] T005 [P] Update `.golangci.yml` to include kube-api-linter settings for `api/tools/` path (jsontags, optionalorrequired, optionalfields, requiredfields, commentstart, nobools, nofloats, integers)

---

## Phase 2: Foundational (API Types, Config Loading, Core Infrastructure)

**Purpose**: Core types, config loading, and shared utilities that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

### API Types (api/tools/v1alpha1/)

- [X] T006 Create package doc with generation markers in `api/tools/v1alpha1/doc.go` — include `+k8s:defaulter-gen=TypeMeta`, `+groupName=mage-common.randomvariable.co.uk`, `+kubebuilder:object:generate=true` markers
- [X] T007 Define all typed structs in `api/tools/v1alpha1/types.go` — ToolConfiguration (with TypeMeta), ToolConfigurationSpec, Tool, ToolSource, SourceType, PlatformOverride, Checksum per data-model.md. Include JSON/YAML struct tags and kube-api-linter markers (`+optional`, `+required`)
- [X] T008 Implement SchemeBuilder and AddToScheme in `api/tools/v1alpha1/register.go` — register GroupVersion `mage-common.randomvariable.co.uk/v1alpha1`, add ToolConfiguration to scheme
- [X] T009 Implement SetDefaults_ToolConfiguration in `api/tools/v1alpha1/defaults.go` — default toolsDir to `hack/bin`, default skipChecksum to false, default allowInsecure to false
- [X] T010 Implement ValidateToolConfiguration in `api/tools/v1alpha1/validation.go` — validate unique tool names, at least one tool, at least one source per tool, mutual exclusion of url/package/path per source type, HTTPS URL enforcement (unless allowInsecure), checksum required for download type (unless skipChecksum), checksum algorithm strength (sha256/sha384/sha512 only, reject md5/sha1), valid Go templates in url/binaryPath, relative toolsDir path
- [X] T011 Run controller-gen to generate `api/tools/v1alpha1/zz_generated.deepcopy.go` (`controller-gen object paths=./api/tools/v1alpha1/`)
- [X] T012 Run defaulter-gen to generate `api/tools/v1alpha1/zz_generated.defaults.go`
- [X] T013 Implement NewSchemeAndCodecs in `api/tools/v1alpha1/scheme/scheme.go` — create scheme, register types, return CodecFactory for YAML deserialization

### Core Infrastructure (tools/)

- [X] T014 [P] Define sentinel errors in `tools/errors.go` — ErrToolNotFound, ErrToolNotInstalled, ErrConfigNotFound, ErrConfigInvalid, ErrRuntimeNotFound, ErrChecksumMismatch, ErrAllSourcesFailed, ErrHTTPSRequired, ErrWeakChecksum, ErrPathTraversal, ErrUnsupportedArchive
- [X] T015 [P] Implement TemplateData construction and Render function in `tools/platform.go` — build TemplateData from version string (parse semver for Major/Minor/Patch, strip v-prefix for VersionNum), resolve OS/Arch through osMap/archMap, execute text/template
- [X] T016 [P] Write tests for platform template rendering in `tools/platform_test.go` — table-driven tests covering v-prefixed versions, bare versions, non-semver (latest), OS/arch mapping, template rendering with all fields
- [X] T017 [P] Implement ToolsDir construction and PATH helpers in `tools/path.go` — resolve `<toolsDir>/<GOOS>/<GOARCH>/` path, PrependToPath, ensure directory creation with restricted permissions
- [X] T018 [P] Write tests for path management in `tools/path_test.go`
- [X] T019 [P] Implement version check and symlink management in `tools/version.go` — IsCurrentVersion (check symlink target matches versioned binary), SetVersion (create versioned binary, atomic symlink update)
- [X] T020 [P] Write tests for version management in `tools/version_test.go` — test symlink creation, version check, update from old version
- [X] T021 [P] Implement exponential backoff retry wrapper in `tools/retry.go` — wrap `k8s.io/apimachinery/pkg/util/wait.ExponentialBackoff` with config from R-003 (1s initial, factor 2.0, jitter 0.1, 3 steps, 10s cap)
- [X] T022 [P] Write tests for retry logic in `tools/retry_test.go`
- [X] T023 Implement config loading (YAML to typed struct) in `tools/config.go` — use scheme/CodecFactory from T013 to deserialize `.tools.yaml`, apply defaults, run validation, return typed ToolConfiguration
- [X] T024 Write tests for config loading in `tools/config_test.go` — valid config, missing file, malformed YAML, validation failures (duplicate names, missing fields, HTTP URLs, weak checksums)

### Test Utilities

- [X] T025 [P] Create test HTTP server helper in `internal/testutil/httpserver.go` — configurable responses, status codes, redirect chains (HTTPS→HTTP rejection testing), slow responses for timeout testing
- [X] T026 [P] Create test archive creation helpers in `internal/testutil/archives.go` — build tar.gz and zip archives in memory with configurable entries (normal files, path traversal entries, symlinks, nested directories)

**Checkpoint**: Foundation ready — API types registered, config loading works, platform templates render, path/version management functional

---

## Phase 3: US1 + US2 — Install All Tools & Install By Name (P1) [MVP]

**Goal**: Developers install all configured tools with one command, or a single tool by name. Includes multi-source fallback (US5) since it's integral to the install flow.

**Independent Test**: Create a `.tools.yaml` with a Go tool and a download tool, run InstallAll, verify both binaries exist at expected paths with correct versions. Run InstallByName for one tool, verify only that tool installed. Test fallback by configuring unreachable first source.

### Install Type Implementations

- [X] T027 [P] [US1] Implement Go install type in `tools/install_go.go` — `go install <url>@<version>` with CGO_ENABLED=0, RuntimeAvailable check for `go` on PATH, IsInstalled via versioned symlink check
- [X] T028 [P] [US1] Write tests for Go install type in `tools/install_go_test.go`
- [X] T029 [P] [US1] Implement Gem install type in `tools/install_gem.go` — Bundler binstub creation, RuntimeAvailable check for `gem`/`bundle` on PATH, IsInstalled via binstub existence
- [X] T030 [P] [US1] Write tests for Gem install type in `tools/install_gem_test.go`
- [X] T031 [P] [US1] Implement npx install type in `tools/install_npx.go` — npx with pinned version, RuntimeAvailable check for `npx` on PATH, IsInstalled via shim existence
- [X] T032 [P] [US1] Write tests for npx install type in `tools/install_npx_test.go`
- [X] T033 [P] [US1] Implement Cargo install type in `tools/install_cargo.go` — `cargo install` with pinned version, RuntimeAvailable check for `cargo` on PATH, IsInstalled via versioned symlink check
- [X] T034 [P] [US1] Write tests for Cargo install type in `tools/install_cargo_test.go`
- [X] T035 [P] [US1] Implement uvx install type in `tools/install_uvx.go` — uvx with pinned version, RuntimeAvailable check for `uvx` on PATH, IsInstalled via shim existence
- [X] T036 [P] [US1] Write tests for uvx install type in `tools/install_uvx_test.go`

### Download Install Type (archive, checksum, HTTP)

- [X] T037 [P] [US1] Implement archive extraction in `tools/archive.go` — tar.gz and zip support, format detection by extension, Zip Slip prevention (reject `../` and absolute paths per FR-051), symlink attack prevention (reject symlinks pointing outside extraction dir), binaryPath traversal validation (FR-052)
- [X] T038 [P] [US1] Write tests for archive extraction in `tools/archive_test.go` — normal extraction, Zip Slip attack (tar with ../), symlink attack, absolute path rejection, binaryPath traversal rejection, missing binary in archive, raw binary (non-archive)
- [X] T039 [US1] Implement download install type in `tools/install_download.go` — HTTP GET with TeeReader checksum (sha256/sha384/sha512), HTTPS→HTTP redirect rejection (FR-053), checksums file parsing (match artifact filename to hash line), archive extraction or raw binary placement, template URL rendering via platform.go
- [X] T040 [US1] Write tests for download install type in `tools/install_download_test.go` — successful download + checksum verify, checksum mismatch, HTTPS→HTTP redirect rejection, checksums file parsing, tar.gz extraction, zip extraction, raw binary download, retry on transient failure

### Installer Orchestrator

- [X] T041 [US1] Implement installer orchestrator in `tools/installer.go` — InstallAll (parallel across types via errgroup, sequential within type), InstallByName (name lookup + single tool install), multi-source fallback (try sources in priority order per FR-024/FR-025), runtime pre-check (FR-022), continue-on-failure with error collection (FR-020), WithFailFast option (FR-021), idempotent skip (FR-006)
- [X] T042 [US1] Write tests for installer orchestrator in `tools/installer_test.go` — install all with mixed types, install by name (found and not found), multi-source fallback (first fails, second succeeds), all sources fail, idempotent skip, fail-fast mode, parallel execution verification, runtime not found error

**Checkpoint**: US1+US2+US5 complete. `tools.InstallAll()` and `tools.InstallByName()` work with all 6 install types, multi-source fallback, and security protections.

---

## Phase 4: US8 — Run Installed Tools with Observable Output (P1)

**Goal**: Mage target authors execute installed tools with command logging (secrets redacted), real-time output streaming, programmatic capture, and exit code inspection.

**Independent Test**: Install a tool, run it via the runner helper, verify command line is printed, stdout/stderr are streamed and capturable, exit code is returned, secrets are redacted from logged command.

- [X] T043 [P] [US8] Implement RunOption functional options in `tools/runner_options.go` — WithStdout, WithStderr, WithCombinedOutput, WithEnv, WithDir, WithSecrets, WithoutFailOnNonZero, WithLogger per data-model RunOption table
- [X] T044 [US8] Implement tool runner in `tools/runner.go` — resolve binary path from tools dir, print command line with secrets redacted (FR-042/FR-043), stream stdout/stderr in real time via io.MultiWriter (FR-044), capture output into RunResult (FR-045), return error on non-zero exit by default (FR-046), capture wall-clock duration (FR-047)
- [X] T045 [US8] Write tests for tool runner in `tools/runner_test.go` — command line logging, secrets redaction, stdout/stderr streaming and capture, combined output, non-zero exit handling (with and without WithoutFailOnNonZero), duration capture, WithEnv and WithDir

**Checkpoint**: US8 complete. `tools.Run()` executes tools with full observability.

---

## Phase 5: US4 — Local Project Tool Builds (P2)

**Goal**: Project maintainers define tools built from local source (custom code generators) alongside remote tools in the same config.

**Independent Test**: Add a `.tools.yaml` entry with a local path to a Go main package, run install, verify the binary is built and placed in the tools directory.

- [X] T046 [US4] Extend Go install type in `tools/install_go.go` to support local path builds — when ToolSource.path is set, use `go build -o <output> <path>` instead of `go install`, detect source changes for rebuild (compare binary mtime vs source mtime)
- [X] T047 [US4] Write tests for local build support in `tools/install_go_test.go` — local build, rebuild on source change, skip when unchanged

**Checkpoint**: US4 complete. Local and remote Go tools coexist in the same config.

---

## Phase 6: US3 — GitHub Actions Caching (P2)

**Goal**: CI pipelines cache the tools directory, skipping installation when config hasn't changed.

**Independent Test**: Generate a cache key, verify it changes when config changes (tool added, version bumped, platform differs) and stays stable when config is unchanged.

- [X] T048 [US3] Implement cache key derivation in `tools/cachekey.go` — SHA-256 hash of `.tools.yaml` content + GOOS + GOARCH, deterministic output (FR-013)
- [X] T049 [US3] Write tests for cache key derivation in `tools/cachekey_test.go` — stable key for same config, key changes on version bump, key changes on tool add/remove, key changes on platform change
- [X] T050 [US3] Create GitHub Actions composite action or documented workflow pattern for tool caching (FR-015) — cache key generation, `actions/cache@v4` integration, install-on-miss pattern per quickstart.md section 4

**Checkpoint**: US3 complete. CI pipelines can cache tools with deterministic keys.

---

## Phase 7: US6 — Update Tools to Latest Versions (P2)

**Goal**: Developers run an update command that queries upstream registries and bumps versions in `.tools.yaml` without reinstalling.

**Independent Test**: Pin a Go tool at an older version, run update, verify the `.tools.yaml` version field is updated to a newer version.

- [X] T051 [P] [US6] Implement Go/GitHub registry querier in `tools/update.go` — query Go module proxy or GitHub API for latest tag, respect tagPrefix (FR-031), use hashicorp/go-version for semver comparison (R-008)
- [X] T052 [P] [US6] Implement npm registry querier in `tools/update.go` — query npm registry for latest version of npx packages
- [X] T053 [P] [US6] Implement crates.io registry querier in `tools/update.go` — query crates.io API for latest version of Cargo packages
- [X] T054 [P] [US6] Implement PyPI registry querier in `tools/update.go` — query PyPI JSON API for latest version of uvx packages
- [X] T055 [P] [US6] Implement RubyGems registry querier in `tools/update.go` — query RubyGems API for latest version of gem packages
- [X] T056 [US6] Implement UpdateAll and UpdateByName orchestration in `tools/update.go` — query each tool's registry, compare versions, update `.tools.yaml` in-place (version fields only), skip `latest`-pinned and local-path-only tools (FR-032), update checksums URLs when version changes (FR-034)
- [X] T057 [US6] Write tests for update logic in `tools/update_test.go` — version bump detection, tag prefix filtering, skip latest, skip local-only, config file modification, checksums URL update, registry query failure handling (continue for other tools)

**Checkpoint**: US6 complete. `tools.UpdateAll()` and `tools.UpdateByName()` work across all ecosystems.

---

## Phase 8: US7 — Documentation and Examples (P3)

**Goal**: New developers can configure and use tool installation within 5 minutes by following documentation.

**Independent Test**: A new team member follows the setup guide to configure tool installation in a fresh project.

- [X] T058 [P] [US7] Write package-level godoc and examples in `tools/doc.go` — package overview, InstallAll example, Run example
- [X] T059 [P] [US7] Add usage examples in `examples/tools/` — complete Magefile showing tools target, lint target, and CI integration
- [X] T060 [US7] Validate quickstart.md against implementation — verify all code samples compile and commands work as documented

**Checkpoint**: US7 complete. Documentation is accurate and actionable.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Dogfooding, linting, final validation

- [X] T061 Create `.tools.yaml` at repository root for mage-common dogfooding (SC-007) — include golangci-lint, controller-gen, defaulter-gen as tools managed by the system itself
- [X] T062 Run kube-api-linter against `api/tools/v1alpha1/` types and fix any violations
- [X] T063 Run `go test ./...` across all packages and verify all tests pass
- [X] T064 Run golangci-lint across the entire repository and fix any issues
- [X] T065 [P] Verify security requirements: Zip Slip test, HTTPS enforcement test, checksum mismatch test, redirect downgrade test, weak algorithm rejection test

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion — BLOCKS all user stories
- **US1+US2+US5 (Phase 3)**: Depends on Phase 2 completion — this is the MVP
- **US8 (Phase 4)**: Depends on Phase 2 (path management) — can run in PARALLEL with Phase 3
- **US4 (Phase 5)**: Depends on Phase 3 (extends install_go.go)
- **US3 (Phase 6)**: Depends on Phase 2 (config loading) — can run in PARALLEL with Phase 3
- **US6 (Phase 7)**: Depends on Phase 2 (config loading) — can run in PARALLEL with Phase 3
- **US7 (Phase 8)**: Depends on Phases 3-7 (needs working implementation to validate docs)
- **Polish (Phase 9)**: Depends on all previous phases

### User Story Dependencies

- **US1+US2 (P1)**: Depends on Foundational — no other story dependencies
- **US8 (P1)**: Depends on Foundational — independent of US1/US2
- **US4 (P2)**: Depends on US1 (extends Go install type)
- **US3 (P2)**: Depends on Foundational — independent of US1/US2
- **US5 (P2)**: Included in US1 (integral to installer)
- **US6 (P2)**: Depends on Foundational — independent of US1/US2
- **US7 (P3)**: Depends on all functional stories

### Within Each User Story

- Types/models before services
- Services before orchestrators
- Tests alongside implementation (TDD per constitution)
- Verify tests pass before marking complete

### Parallel Opportunities

```
Phase 2 (after T006-T008 types are done):
  T014, T015, T017, T019, T021, T025, T026 — all [P], different files

Phase 3 (all install types are independent):
  T027, T029, T031, T033, T035, T037 — all [P], different files
  (T039 depends on T037 archive.go)
  (T041 depends on all install types)

Phase 4 can run in PARALLEL with Phase 3:
  T043 — no dependency on install types

Phase 6 + Phase 7 can run in PARALLEL with Phase 3:
  T048 (cache key) — only needs config loading
  T051-T055 (registry queriers) — all [P], different registries
```

---

## Parallel Example: Phase 3 Install Types

```
# Launch all install type implementations in parallel:
Task: T027 "Go install type in tools/install_go.go"
Task: T029 "Gem install type in tools/install_gem.go"
Task: T031 "npx install type in tools/install_npx.go"
Task: T033 "Cargo install type in tools/install_cargo.go"
Task: T035 "uvx install type in tools/install_uvx.go"
Task: T037 "Archive extraction in tools/archive.go"

# Then sequentially (dependencies):
Task: T039 "Download install type" (depends on T037)
Task: T041 "Installer orchestrator" (depends on all install types)
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T026)
3. Complete Phase 3: US1+US2 Install All & By Name (T027-T042)
4. **STOP and VALIDATE**: Run InstallAll with a multi-tool config, verify all binaries installed
5. This delivers the core value proposition

### Incremental Delivery

1. Setup + Foundational (Phases 1-2) -> Infrastructure ready
2. US1+US2+US5 (Phase 3) -> Core install works (MVP!)
3. US8 (Phase 4) -> Tool runner works -> can build Mage targets
4. US4 (Phase 5) -> Local builds work -> complete install story
5. US3 (Phase 6) -> CI caching works -> fast CI
6. US6 (Phase 7) -> Updates work -> maintenance story
7. US7 (Phase 8) -> Docs complete -> onboarding story
8. Polish (Phase 9) -> Production ready

### Parallel Team Strategy

With multiple developers after Phase 2 completes:

- **Developer A**: Phase 3 (US1+US2 — install types + orchestrator)
- **Developer B**: Phase 4 (US8 — tool runner) + Phase 6 (US3 — cache key)
- **Developer C**: Phase 7 (US6 — update command)

---

## Summary

| Phase | Story | Priority | Tasks | Parallel |
|-------|-------|----------|-------|----------|
| 1 Setup | — | — | T001-T005 (5) | 2 |
| 2 Foundational | — | — | T006-T026 (21) | 12 |
| 3 Install | US1+US2+US5 | P1/P2 | T027-T042 (16) | 10 |
| 4 Runner | US8 | P1 | T043-T045 (3) | 1 |
| 5 Local Builds | US4 | P2 | T046-T047 (2) | 0 |
| 6 Caching | US3 | P2 | T048-T050 (3) | 0 |
| 7 Update | US6 | P2 | T051-T057 (7) | 5 |
| 8 Docs | US7 | P3 | T058-T060 (3) | 2 |
| 9 Polish | — | — | T061-T065 (5) | 1 |
| **Total** | | | **65** | **33** |

---

## Notes

- [P] tasks = different files, no dependencies — safe for parallel execution
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- TDD: write tests alongside implementation per constitution Principle VI
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
