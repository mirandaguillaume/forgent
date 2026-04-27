# Story 6.2: File Access Control from Skill YAML

Status: done

## Story

As a **skill author**,
I want to declare file read/write permissions in my skill YAML security section,
so that the runtime prevents the LLM from accessing files outside the declared scope — enforced by code, not by prompting.

## Acceptance Criteria

1. A new `FileAccessPolicy` struct is defined in a shared package (`pkg/sandbox/`) containing `ReadGlobs`, `WriteGlobs`, and `DenyGlobs` fields (string slices)
2. The `SecurityFacet` in `pkg/model/skill.go` gains a `FileAccess *FileAccessConfig` field that maps the YAML `security.file_access` section to `read`, `write`, and `deny` glob lists
3. A `FileAccessMiddleware` function returns a `dag.Middleware` that intercepts tool calls involving file paths (`read_file`/`file_read`, `write_file`/`file_write`, `list_directory`, `file_delete`) and validates them against the `FileAccessPolicy`
4. When the LLM step tries to **read** a file that does not match any `ReadGlobs` pattern, the read is blocked and the handler receives a descriptive error: `"file access denied: read <path> not permitted by policy (allowed: [globs])"`
5. When the LLM step tries to **write** a file that does not match any `WriteGlobs` pattern, the write is blocked with a descriptive error
6. When the LLM step tries to read OR write a file matching any `DenyGlobs` pattern, the action is blocked **regardless of allow rules** (deny takes precedence)
7. When no `security.file_access` is configured in the skill YAML (nil `FileAccess`), the middleware is a no-op pass-through (zero overhead)
8. The core glob-matching and policy-checking logic from `internal/bench/experiment/sandbox.go` is extracted to `pkg/sandbox/` so both the experiment harness and production middleware can share it
9. The `internal/bench/experiment/sandbox.go` is refactored to delegate to `pkg/sandbox/` for policy checking and glob matching (no duplication of glob logic)
10. All existing tests pass without modification (`go test ./... -count=1 -race`)
11. The middleware works as both a pre-middleware (blocks file operations before they execute) and returns errors that propagate through the DAG retry mechanism

## Tasks / Subtasks

- [ ] Task 1: Create shared sandbox package `pkg/sandbox/` (AC: #1, #8)
  - [ ] 1.1: Create `pkg/sandbox/policy.go` with `FileAccessPolicy` struct:
    ```go
    type FileAccessPolicy struct {
        ReadGlobs  []string // globs for allowed reads; empty = allow all
        WriteGlobs []string // globs for allowed writes; empty = allow all
        DenyGlobs  []string // globs that are always denied (takes precedence)
    }
    ```
  - [ ] 1.2: Create `pkg/sandbox/glob.go` — extract `globMatch` function from `internal/bench/experiment/sandbox.go` into this package. The function is `func GlobMatch(pattern, path string) bool` (exported). Exact same logic: supports `**` for recursive directory matching in any position.
  - [ ] 1.3: Create `pkg/sandbox/check.go` with `CheckAccess` function:
    ```go
    // CheckAccess validates a file path against a FileAccessPolicy for a given action.
    // action is "read", "write", or "delete".
    // Returns ("", true) if allowed, or (reason, false) if denied.
    func CheckAccess(policy *FileAccessPolicy, path string, action string) (reason string, allowed bool)
    ```
    Logic:
    - If `policy` is nil, return ("", true) — no policy = allow all
    - Check DenyGlobs first: if path matches any deny glob, return denied (deny always wins)
    - For action "read": if ReadGlobs is non-empty, path must match at least one ReadGlob; if ReadGlobs is empty, allow
    - For action "write" or "delete": if WriteGlobs is non-empty, path must match at least one WriteGlob; if WriteGlobs is empty, allow
  - [ ] 1.4: Create `pkg/sandbox/glob_test.go` — migrate existing glob tests from `internal/bench/experiment/sandbox_test.go` (TestGlobMatch_MidPattern equivalents) plus new cases for the exported `GlobMatch` function
  - [ ] 1.5: Create `pkg/sandbox/check_test.go` — tests for `CheckAccess`:
    - `TestCheckAccess_NilPolicy` — nil policy allows everything
    - `TestCheckAccess_ReadAllowed` — path matching ReadGlobs is allowed
    - `TestCheckAccess_ReadDenied` — path not matching any ReadGlob is denied
    - `TestCheckAccess_WriteAllowed` — path matching WriteGlobs is allowed
    - `TestCheckAccess_WriteDenied` — path not matching any WriteGlob is denied
    - `TestCheckAccess_DenyTakesPrecedence` — path in both ReadGlobs and DenyGlobs is denied
    - `TestCheckAccess_EmptyReadGlobs_AllowAll` — no ReadGlobs = no restriction on reads
    - `TestCheckAccess_EmptyWriteGlobs_AllowAll` — no WriteGlobs = no restriction on writes
    - `TestCheckAccess_DeleteUsesWriteGlobs` — delete action checks against WriteGlobs

- [ ] Task 2: Refactor `internal/bench/experiment/sandbox.go` to use `pkg/sandbox/` (AC: #9, #10)
  - [ ] 2.1: Replace the `globMatch` function in `internal/bench/experiment/sandbox.go` with a call to `sandbox.GlobMatch` from `pkg/sandbox/`. The local `globMatch` function is either removed entirely or becomes a thin wrapper: `func globMatch(p, path string) bool { return sandbox.GlobMatch(p, path) }`. Prefer the thin wrapper to minimize test churn.
  - [ ] 2.2: Verify all existing experiment tests pass: `go test ./internal/bench/experiment/... -count=1 -race`. No test changes should be needed — the wrapper preserves the exact same API.

- [ ] Task 3: Add `FileAccess` field to `SecurityFacet` in `pkg/model/skill.go` (AC: #2)
  - [ ] 3.1: Define `FileAccessConfig` struct in `pkg/model/skill.go`:
    ```go
    // FileAccessConfig declares file access permissions for a skill.
    type FileAccessConfig struct {
        Read []string `yaml:"read,omitempty"` // glob patterns for allowed reads
        Write []string `yaml:"write,omitempty"` // glob patterns for allowed writes
        Deny []string `yaml:"deny,omitempty"` // glob patterns always denied
    }
    ```
  - [ ] 3.2: Add `FileAccess *FileAccessConfig` field to `SecurityFacet`:
    ```go
    type SecurityFacet struct {
        Filesystem AccessLevel      `yaml:"filesystem"`
        Network    NetworkAccess    `yaml:"network"`
        Secrets    []string         `yaml:"secrets"`
        Sandbox    SandboxType      `yaml:"sandbox,omitempty"`
        FileAccess *FileAccessConfig `yaml:"file_access,omitempty"`
    }
    ```
  - [ ] 3.3: Add a helper method `ToPolicy` on `FileAccessConfig` that converts to `sandbox.FileAccessPolicy`:
    ```go
    func (f *FileAccessConfig) ToPolicy() *sandbox.FileAccessPolicy {
        if f == nil {
            return nil
        }
        return &sandbox.FileAccessPolicy{
            ReadGlobs:  f.Read,
            WriteGlobs: f.Write,
            DenyGlobs:  f.Deny,
        }
    }
    ```
  - [ ] 3.4: Verify existing model tests pass: `go test ./pkg/model/... -count=1 -race`. The new field is optional (pointer + omitempty), so existing YAML unmarshaling is unaffected.

- [ ] Task 4: Create `FileAccessMiddleware` in `pkg/sandbox/middleware.go` (AC: #3, #4, #5, #6, #7, #11)
  - [ ] 4.1: Create `pkg/sandbox/middleware.go` that imports `pkg/dag` for the `Middleware` and `MiddlewareFunc` types (from Story 6-1). Define:
    ```go
    // FileAccessMiddleware returns a dag.Middleware that enforces file access policy
    // on tool call inputs. It inspects the inputs map for keys that indicate file
    // operations (tool_name, file_path) and validates them against the policy.
    //
    // Convention: the node's Run function receives inputs with:
    //   - "tool_name" (string): the tool being invoked (e.g. "file_read", "write_file")
    //   - "file_path" (string): the file path being operated on
    //
    // If the policy is nil, returns a no-op middleware (passes through to next).
    func FileAccessMiddleware(policy *FileAccessPolicy) dag.Middleware
    ```
  - [ ] 4.2: Implementation of `FileAccessMiddleware`:
    - If `policy` is nil, return a pass-through middleware: `func(ctx, inputs, next) { return next(ctx, inputs) }`
    - Extract `tool_name` and `file_path` from `inputs` map (as string values)
    - If neither key is present, pass through to `next` (not a file operation)
    - Determine the action from the tool name:
      - `"file_read"`, `"read_file"` → action = `"read"`
      - `"file_write"`, `"write_file"` → action = `"write"`
      - `"file_delete"`, `"delete_file"` → action = `"delete"`
      - `"list_directory"`, `"list_dir"` → action = `"read"` (listing is a read operation)
      - Any other tool_name → pass through (not a file operation this middleware handles)
    - Call `CheckAccess(policy, filePath, action)`
    - If denied: return `nil, fmt.Errorf("file access denied: %s %q not permitted by policy (%s)", action, filePath, reason)`
    - If allowed: return `next(ctx, inputs)`
  - [ ] 4.3: Also support inspecting outputs (post-middleware behavior). After `next(ctx, inputs)` returns, if the outputs contain `"file_path"` and `"tool_name"`, validate the output path too. This catches cases where a tool reads one path but returns content from another. If the output file_path is denied, return an error.
    - IMPORTANT: The primary enforcement is pre-middleware (blocking before execution). Post-middleware is a secondary safety net. Both happen within the same middleware function (check inputs before `next`, check outputs after `next`).

- [ ] Task 5: Unit tests for `FileAccessMiddleware` in `pkg/sandbox/middleware_test.go` (AC: #3, #4, #5, #6, #7, #11)
  - [ ] 5.1: `TestFileAccessMiddleware_NilPolicy` — nil policy passes through; handler is called; outputs returned unchanged
  - [ ] 5.2: `TestFileAccessMiddleware_ReadAllowed` — inputs with tool_name="file_read", file_path="src/main.go"; ReadGlobs=["src/**"]; handler is called
  - [ ] 5.3: `TestFileAccessMiddleware_ReadDenied` — inputs with tool_name="file_read", file_path="secrets/.env"; ReadGlobs=["src/**"]; handler NOT called; error contains "file access denied" and "read"
  - [ ] 5.4: `TestFileAccessMiddleware_WriteAllowed` — inputs with tool_name="file_write", file_path="src/new.go"; WriteGlobs=["src/**"]; handler is called
  - [ ] 5.5: `TestFileAccessMiddleware_WriteDenied` — inputs with tool_name="file_write", file_path="etc/passwd"; WriteGlobs=["src/**"]; handler NOT called; error returned
  - [ ] 5.6: `TestFileAccessMiddleware_DenyOverridesAllow` — DenyGlobs=["**/.env"]; ReadGlobs=["**"]; file_path=".env"; handler NOT called; reason mentions deny glob
  - [ ] 5.7: `TestFileAccessMiddleware_DeleteChecksWriteGlobs` — inputs with tool_name="file_delete"; uses WriteGlobs for validation
  - [ ] 5.8: `TestFileAccessMiddleware_ListDirectoryChecksReadGlobs` — inputs with tool_name="list_directory"; uses ReadGlobs for validation
  - [ ] 5.9: `TestFileAccessMiddleware_NonFileToolPassesThrough` — inputs with tool_name="search_code"; no file_path; handler is called regardless of policy
  - [ ] 5.10: `TestFileAccessMiddleware_NoToolNamePassesThrough` — inputs without tool_name key; handler is called
  - [ ] 5.11: `TestFileAccessMiddleware_EmptyGlobsAllowAll` — ReadGlobs=nil, WriteGlobs=nil; all reads and writes pass through
  - [ ] 5.12: `TestFileAccessMiddleware_ErrorMessageDescriptive` — verify error message includes: action, path, and the reason from CheckAccess
  - [ ] 5.13: `TestFileAccessMiddleware_WithDAGRetry` — integration test: node with FileAccessMiddleware + MaxRetries=1; first call has denied path in inputs; verify error propagates and triggers retry in DAG

- [ ] Task 6: Integration tests with DAG execution in `pkg/sandbox/middleware_integration_test.go` (AC: #3, #11)
  - [ ] 6.1: `TestFileAccessMiddleware_DAG_BlockedNodePropagatesError` — single-node DAG with FileAccessMiddleware; node inputs include denied file_path; DAG.Execute returns error wrapping "file access denied"
  - [ ] 6.2: `TestFileAccessMiddleware_DAG_AllowedNodeExecutes` — single-node DAG with FileAccessMiddleware; node inputs include allowed file_path; DAG.Execute succeeds
  - [ ] 6.3: `TestFileAccessMiddleware_DAG_MultiNodePipeline` — 3-node pipeline; only middle node has FileAccessMiddleware; first and last nodes have no middleware; verify middleware only affects middle node
  - [ ] 6.4: `TestFileAccessMiddleware_DAG_WithRetryAndPolicyChange` — node with MaxRetries=1 and FileAccessMiddleware; handler changes the file_path in its output on second call to an allowed path; verify retry succeeds (note: this tests the interaction between middleware and retry, since the middleware re-evaluates on each attempt)

- [ ] Task 7: Add `FileAccessConfig` YAML round-trip test in `pkg/model/skill_test.go` (AC: #2)
  - [ ] 7.1: `TestSkillBehavior_FileAccessYAML` — unmarshal a skill YAML with `security.file_access` section:
    ```yaml
    security:
      filesystem: read-write
      file_access:
        read:
          - "src/**/*.go"
          - "docs/**"
        write:
          - "src/**/*.go"
          - "tmp/**"
        deny:
          - "**/.env"
          - "**/credentials*"
          - "**/*.key"
    ```
    Verify: `FileAccess` is non-nil, `Read` has 2 entries, `Write` has 2 entries, `Deny` has 3 entries
  - [ ] 7.2: `TestSkillBehavior_FileAccessYAML_Absent` — unmarshal a skill YAML without `file_access`; verify `FileAccess` is nil
  - [ ] 7.3: `TestFileAccessConfig_ToPolicy` — test the `ToPolicy()` conversion: non-nil config returns correct policy; nil config returns nil policy

- [ ] Task 8: Verify all existing tests pass (AC: #10)
  - [ ] 8.1: Run `go test ./pkg/sandbox/... -count=1 -race` — all new tests pass
  - [ ] 8.2: Run `go test ./pkg/model/... -count=1 -race` — existing + new model tests pass
  - [ ] 8.3: Run `go test ./pkg/dag/... -count=1 -race` — existing DAG tests unaffected
  - [ ] 8.4: Run `go test ./internal/bench/experiment/... -count=1 -race` — experiment tests pass after refactor
  - [ ] 8.5: Run `go test ./... -count=1 -race` — full suite passes

## Dev Notes

### Architecture References

- **ADR-1 (architecture.md):** Doctor embedding strategy — `BuildDAG()` in `internal/doctor/orchestrator.go` constructs the DAG. File access middleware will be applied to doctor nodes in a future story or when doctor itself declares `security.file_access` in its skill YAML.
- **ADR-4 (architecture.md):** Doctor DAG topology — 4 nodes across 3 layers. The file access middleware can be attached to any node via the `Middlewares` field (from Story 6-1).
- **Epic 6 rationale (epics.md):** "Experiment results show text beats runtime on guardrails without programmatic enforcement. The runtime's value is not in 'better LLM responses' but in 'programmatically preventing bad actions.'" File access control is the first concrete enforcement mechanism.
- **Epic 5 retro lesson #1:** "Framework != Evidence." This story builds a real enforcement mechanism, not just a framework. The middleware produces measurable outcomes (blocked vs allowed actions) that Story 6-6 will use for evidence.

### Key Design Decisions

**1. Shared package at `pkg/sandbox/` not `internal/sandbox/`**

The sandbox logic is placed in `pkg/sandbox/` (public API) because:
- It is consumed by `pkg/dag` middleware (which is in `pkg/`)
- It is consumed by `internal/bench/experiment/` (which currently has its own copy)
- Skill authors building custom Forgent runtimes may need to construct `FileAccessPolicy` directly
- The API is small and stable: `FileAccessPolicy`, `GlobMatch`, `CheckAccess`, `FileAccessMiddleware`

**2. Thin wrapper in experiment package preserves backward compatibility**

Instead of rewriting all experiment tests, `internal/bench/experiment/sandbox.go` keeps its local `globMatch` function as a one-line wrapper around `sandbox.GlobMatch`. The `SandboxFS` struct continues to exist in the experiment package because it has experiment-specific features (traps, audit trail, in-memory virtual filesystem) that are not needed by the production middleware. Only the glob matching and policy checking logic are shared.

**3. Middleware inspects `tool_name` and `file_path` keys in inputs map**

The convention for passing tool call information through DAG nodes is via the `inputs` map. The middleware looks for:
- `"tool_name"` (string) — identifies the tool being called
- `"file_path"` (string) — the target file path

This is a convention, not a schema enforcement. Nodes that don't put these keys in their inputs are transparently passed through. This means the middleware is safe to attach to any node — it only activates when file operation keys are present.

This design mirrors how the experiment runner already works: `ExecuteToolWithPolicy` checks tool calls by inspecting `ToolCall.Name` and `ToolCall.Args["path"]`. The middleware generalizes this to the DAG inputs convention.

**4. Deny globs always take precedence**

The check order is: deny first, then allow. This matches the existing `SandboxFS.checkDenied` logic and is the standard security practice (deny overrides allow). A path matching a deny glob is rejected even if it also matches an allow glob.

**5. Empty globs = no restriction for that action**

If `ReadGlobs` is empty/nil, all reads are allowed (no restriction). If `WriteGlobs` is empty/nil, all writes are allowed. Only `DenyGlobs` restrict access when the allow globs are empty. This is "open by default, restricted when declared" — consistent with the existing `SandboxFS` behavior where an empty `AllowedGlobs` means no path restriction.

**6. `file_delete` uses WriteGlobs**

Deleting a file is a destructive write operation. It is validated against `WriteGlobs`, not `ReadGlobs`. This matches the existing experiment sandbox where `Delete` checks the same policy as `Write`.

**7. Middleware does NOT create a SandboxFS**

The `FileAccessMiddleware` is a pure policy check — it validates paths against globs and returns errors. It does NOT manage an in-memory filesystem, audit trail, or trap files. Those features remain in the experiment package's `SandboxFS` for experiment-specific needs. The production middleware is lightweight: no allocations on the hot path (just glob matching).

**8. Post-middleware output check is secondary**

The primary enforcement is pre-middleware: before the handler runs, check if the requested file operation is allowed. The optional post-middleware check validates that the handler's output doesn't reference denied paths. This is a defense-in-depth measure; the pre-check is what matters for security.

### Dependencies

- **Story 6-1 (middleware pipeline):** MUST be completed first. This story depends on `dag.Middleware`, `dag.MiddlewareFunc`, and the `Middlewares []Middleware` field on `Node`. If Story 6-1 is not yet merged, the middleware code (`pkg/sandbox/middleware.go` and its tests) cannot compile.
- **Upstream:** `pkg/dag` (Story 6-1 adds middleware types), `pkg/model` (this story extends it)
- **Downstream stories that consume this:**
  - Story 6-6: Harness Experiment — will construct `FileAccessPolicy` from task definitions and attach `FileAccessMiddleware` to DAG nodes
  - Future: Doctor DAG can declare `security.file_access` in its skill YAML and get automatic file access enforcement

### What NOT to Build

- Do NOT create a full in-memory virtual filesystem in `pkg/sandbox/` — the VirtualFS (SandboxFS with file storage, traps, audit trail) stays in `internal/bench/experiment/` where it serves experiment-specific needs
- Do NOT modify `internal/doctor/orchestrator.go` to add file access middleware to doctor nodes — that is a future concern when doctor's skill YAMLs declare file access policies
- Do NOT modify `internal/bench/experiment/runtime_runner.go` — experiment integration with the new middleware comes in Story 6-6
- Do NOT add network access control — that is a separate concern for a future story
- Do NOT add per-step policy overrides in the middleware — the middleware receives a single `FileAccessPolicy` per node. Per-step policies (like the experiment's `StepSandboxPolicy`) are a higher-level concern handled by the caller when constructing the DAG
- Do NOT add audit trail / logging to the middleware — observability middleware is a separate concern (deferred)
- Do NOT modify the `SandboxPolicy` struct in `internal/bench/experiment/sandbox.go` — it has its own richer fields (MaxFileSize, ReadOnly, DeniedWritePatterns, traps) that go beyond what `FileAccessPolicy` covers
- Do NOT add middleware attachment logic (reading skill YAML and constructing middleware on nodes) — that is the DAG builder's responsibility, not this story. This story provides the middleware; consumers attach it.

### Project Structure

New files:
- `pkg/sandbox/policy.go` — `FileAccessPolicy` struct
- `pkg/sandbox/glob.go` — `GlobMatch` function (extracted from experiment sandbox)
- `pkg/sandbox/check.go` — `CheckAccess` function
- `pkg/sandbox/middleware.go` — `FileAccessMiddleware` function (returns `dag.Middleware`)
- `pkg/sandbox/glob_test.go` — glob matching tests
- `pkg/sandbox/check_test.go` — access checking tests
- `pkg/sandbox/middleware_test.go` — middleware unit tests
- `pkg/sandbox/middleware_integration_test.go` — DAG integration tests

Modified files:
- `pkg/model/skill.go` — add `FileAccessConfig` struct and `FileAccess` field to `SecurityFacet`
- `pkg/model/skill_test.go` — add YAML round-trip tests for `file_access`
- `internal/bench/experiment/sandbox.go` — replace local `globMatch` with wrapper around `sandbox.GlobMatch`

### Existing Code Reference

**`SecurityFacet` struct (pkg/model/skill.go:152-158):**
```go
type SecurityFacet struct {
    Filesystem AccessLevel   `yaml:"filesystem"`
    Network    NetworkAccess `yaml:"network"`
    Secrets    []string      `yaml:"secrets"`
    Sandbox    SandboxType   `yaml:"sandbox,omitempty"`
}
```

**`globMatch` in experiment sandbox (internal/bench/experiment/sandbox.go:252-305):**
```go
func globMatch(pattern, path string) bool {
    if !strings.Contains(pattern, "**") {
        matched, err := filepath.Match(pattern, path)
        if err == nil && matched {
            return true
        }
        return pattern == path
    }
    // ... recursive ** matching logic ...
}
```

**`checkDenied` in experiment sandbox (internal/bench/experiment/sandbox.go:160-188):**
```go
func (s *SandboxFS) checkDenied(path string) (string, bool) {
    // Trap check first
    if s.traps[path] {
        return "trap file access", false
    }
    // Denied globs check
    for _, pattern := range s.policy.DeniedGlobs {
        if globMatch(pattern, path) {
            return fmt.Sprintf("denied by glob %q", pattern), false
        }
    }
    // Allowed globs check
    if len(s.policy.AllowedGlobs) > 0 {
        allowed := false
        for _, pattern := range s.policy.AllowedGlobs {
            if globMatch(pattern, path) {
                allowed = true
                break
            }
        }
        if !allowed {
            return "not in allowed paths", false
        }
    }
    return "", true
}
```

**Middleware types from Story 6-1 (to be created in pkg/dag/middleware.go):**
```go
type MiddlewareFunc func(ctx context.Context, inputs map[string]any) (map[string]any, error)
type Middleware func(ctx context.Context, inputs map[string]any, next MiddlewareFunc) (map[string]any, error)
```

**`Node` struct after Story 6-1 (pkg/dag/dag.go — with Middlewares field added):**
```go
type Node struct {
    ID         string
    Kind       NodeKind
    Consumes   []string
    Produces   []string
    Run        func(ctx context.Context, inputs map[string]any) (map[string]any, error)
    MaxRetries int
    Timeout    time.Duration
    Config     *NodeConfig
    Middlewares []Middleware  // added by Story 6-1
}
```

**YAML example for skill with file access control:**
```yaml
skill: code-reviewer
version: "1.0"
context:
  consumes: [source_code, review_criteria]
  produces: [code_review]
strategy:
  tools: [file_read, file_write]
  approach: llm
security:
  filesystem: read-write
  file_access:
    read:
      - "src/**/*.go"
      - "pkg/**/*.go"
      - "internal/**/*.go"
      - "docs/**"
    write:
      - "src/**/*.go"
      - "tmp/**"
    deny:
      - "**/.env"
      - "**/credentials*"
      - "**/*.key"
      - "**/*.pem"
```

### Implementation Order

Tasks MUST be executed in this order due to dependencies:

1. **Task 1** (pkg/sandbox/) — foundation; no dependencies
2. **Task 2** (refactor experiment sandbox) — depends on Task 1
3. **Task 3** (model extension) — depends on Task 1 (for the ToPolicy helper)
4. **Task 4** (middleware) — depends on Task 1 AND Story 6-1 being merged
5. **Tasks 5-6** (middleware tests) — depend on Task 4
6. **Task 7** (model tests) — depends on Task 3
7. **Task 8** (full regression) — depends on all above

If Story 6-1 is not yet merged when development starts, Tasks 1-3 and Task 7 can be implemented and tested independently. Task 4-6 must wait for the `dag.Middleware` type to exist.

## Testing Strategy

**Unit tests (Tasks 1, 5, 7):** Pure function tests for glob matching, access checking, and middleware behavior in isolation. No DAG construction. Test policy logic comprehensively with table-driven tests.

**Refactor validation (Task 2):** Verify that extracting `globMatch` to a shared package does not break the 50+ existing experiment tests. Run the full experiment test suite.

**Integration tests (Task 6):** Full DAG construction and execution with `FileAccessMiddleware` configured on nodes. Tests verify that:
- Blocked file access propagates as DAG execution errors
- Allowed file access passes through transparently
- Middleware interacts correctly with retry mechanism
- Per-node middleware isolation (middleware on one node doesn't affect others)

**Regression tests (Task 8):** All existing tests must pass unchanged. Run full suite with `-race`.

**Test patterns to follow:**
- `require` for preconditions, `assert` for values (testify convention from project)
- Table-driven tests for glob matching and policy checking (many cases, compact format)
- Atomic counters for verifying handler invocation counts
- Section headers with ASCII art comments (project convention: `// --- Section ---`)

## References

- [Source: _bmad-output/planning-artifacts/epics.md#Story6.2] — Story acceptance criteria from epic
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-1] — Embedded YAML + runtime DAG
- [Source: _bmad-output/implementation-artifacts/6-1-middleware-pipeline.md] — Middleware type signatures, integration point
- [Source: internal/bench/experiment/sandbox.go] — Existing glob matching, policy checking, SandboxFS
- [Source: internal/bench/experiment/sandbox_test.go] — Existing 50+ sandbox tests
- [Source: internal/bench/experiment/tools.go] — Tool call execution, ExecuteToolWithPolicy
- [Source: internal/bench/experiment/runtime_runner.go] — buildSandboxFromWorkspace, DAG construction
- [Source: pkg/model/skill.go] — SecurityFacet struct, SkillBehavior
- [Source: pkg/dag/dag.go] — Node struct, DAG construction
- [Source: pkg/dag/execute.go] — runWithRetry, middleware integration point
- [Source: _bmad-output/implementation-artifacts/epic-5-retro-2026-04-15.md] — "Framework != Evidence" lesson

## Dev Agent Record

### Agent Model Used

(to be filled by dev agent)

### Debug Log References

(to be filled by dev agent)

### Completion Notes List

(to be filled by dev agent)

### File List

(to be filled by dev agent)
