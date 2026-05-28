# Story 10-5: HTTP Proxy Network Sandbox

**Epic:** 10 — Node-Level Behavioral Contracts
**Status:** done

## Story

As a **developer defining a Forgent task**,
I want to declare a network policy at the step level (`allowed_domains` or `forbidden_domains`),
So that the runtime blocks unauthorized outbound HTTP calls via an injected proxy before they reach the network.

## Acceptance Criteria

**AC1:** Given a step declares `network_policy: { forbidden_domains: ["api.stripe.com", "*.internal"] }`,
When any HTTP client routed through the injected proxy attempts a request to a forbidden domain,
Then the proxy intercepts and returns `403 Forbidden: domain not permitted for this step`,
And the violation is recorded in the audit trail with `Allowed: false` and a `Reason` containing the matched forbidden domain pattern.

**AC2:** Given a step declares `network_policy: { allowed_domains: ["api.github.com"] }`,
When a request targets a domain not in `allowed_domains`,
Then the proxy blocks it with `403 Forbidden: domain not in allowed_domains`,
And requests to listed domains are forwarded normally and return the upstream response.

**AC3:** Given a step has no `network_policy` declared,
When the runtime executes,
Then no proxy is started — full backward compatibility (NFR40),
And no `HTTP_PROXY` / `HTTPS_PROXY` env var is set for that step.

**AC4:** Given the proxy is active for a step,
When the step's `runFn` executes,
Then `HTTP_PROXY` and `HTTPS_PROXY` env vars are set to the proxy's listen URL (e.g. `http://127.0.0.1:<ephemeral-port>`),
And the proxy listens on a local ephemeral port (TCP `127.0.0.1:0` → kernel-assigned port),
And the proxy is shut down at the end of the step (deferred close in `runFn`).

**AC5:** Given both `allowed_domains` and `forbidden_domains` are declared on a step,
When a request targets a domain that is BOTH in `allowed_domains` AND matches a `forbidden_domains` pattern,
Then `forbidden_domains` wins (deny-takes-precedence),
And the violation is recorded with the forbidden pattern in the reason.

**AC6:** Given an unknown field appears in `network_policy` (e.g. `network_policy: { allow_globs: [...] }` typo),
When `Validate()` runs,
Then a clear error names the offending field and lists the valid keys (`allowed_domains`, `forbidden_domains`).

**AC7:** Given a domain pattern includes a glob (`*.internal`),
When the proxy matches a request host against the pattern,
Then `*.internal` matches `api.internal` and `db.cluster.internal` (suffix match — same semantics as `pkg/sandbox.GlobMatch` for path globs adapted to dot-separated host labels),
And `*.internal` does NOT match `internal` (no leading-dot label is not the same as the suffix).

**AC8:** Given `go test ./... -count=1` and `go test -race ./internal/bench/experiment/... ./pkg/sandbox/...`,
Then all existing tests pass without modification (race-clean),
And the new tests cover proxy lifecycle, domain matching, env var injection, audit recording, deny-precedence, and backward compatibility.

## Tasks/Subtasks

- [ ] T1: Network policy data model + matcher (`pkg/sandbox/network.go`)
  - [ ] T1.1: Define `NetworkPolicy` struct in `pkg/sandbox/network.go` parallel to `FileAccessPolicy`:
        ```go
        type NetworkPolicy struct {
            AllowedDomains   []string // exact host or glob (e.g. "api.github.com", "*.internal")
            ForbiddenDomains []string // takes precedence over AllowedDomains
        }
        ```
  - [ ] T1.2: Implement `(p *NetworkPolicy) CheckHost(host string) (allowed bool, reason string)`:
        - If `host` matches any `ForbiddenDomains` pattern → `(false, "domain matches forbidden pattern: <pattern>")`
        - Else if `AllowedDomains` is empty → `(true, "")` (default allow when no allow-list configured)
        - Else if `host` matches any `AllowedDomains` pattern → `(true, "")`
        - Else → `(false, "domain not in allowed_domains")`
  - [ ] T1.3: Implement `hostMatch(pattern, host string) bool` adapting `GlobMatch` semantics for
        dot-separated host labels:
        - Exact match: `pattern == host`
        - Suffix wildcard: `*.foo.bar` matches `x.foo.bar` and `a.b.foo.bar`, NOT `foo.bar`
        Re-use `pkg/sandbox.GlobMatch` if its semantics fit; otherwise add a host-specific
        helper next to it. Document the divergence in a doc comment if any.
  - [ ] T1.4: Tests in `pkg/sandbox/network_test.go`:
        - Exact host match
        - Glob suffix match (`*.internal` matches `api.internal` AND `a.b.internal`)
        - Glob does not match the bare domain (`*.internal` does NOT match `internal`)
        - Deny precedence (host in both allowed and forbidden → forbidden wins)
        - Empty policy → all hosts allowed (backward compat)
        - Empty allow + non-empty forbid → all hosts allowed except the forbidden ones
        - Empty forbid + non-empty allow → only listed hosts allowed

- [ ] T2: HTTP proxy server (`pkg/sandbox/network.go` or new file)
  - [ ] T2.1: Implement `NewProxyServer(policy *NetworkPolicy, audit AuditFunc) (*ProxyServer, error)`:
        - Listen on `127.0.0.1:0` so the kernel assigns an ephemeral port
        - Store the listen address and a shutdown channel
        - `audit AuditFunc` is a callback `func(host string, allowed bool, reason string)` so the
          caller can record violations in the experiment audit trail without coupling this
          package to `internal/bench/experiment/`
  - [ ] T2.2: `ProxyServer.URL() string` returns `http://127.0.0.1:<port>` for env var injection.
  - [ ] T2.3: `ProxyServer.Close() error` stops the listener and drains in-flight handlers.
  - [ ] T2.4: HTTP handler implementation:
        - For `CONNECT` (HTTPS tunnel): parse `Host:Port` from the request URL host, run
          `policy.CheckHost(host)`, return `403 Forbidden: <reason>` on deny, otherwise
          establish the tunnel via `net.Dial("tcp", host+":"+port)` + `io.Copy` bidirectional.
          Always invoke `audit(host, allowed, reason)` whether the decision was allow or deny.
        - For plain HTTP (`GET /...` with absolute URL): extract host from `r.URL.Host`,
          check the policy, on allow forward via `http.DefaultTransport.RoundTrip` and copy
          the response back; on deny return 403 with reason.
        - On any internal error (`Dial` fails, RoundTrip fails) → return `502 Bad Gateway` with a
          short reason; do NOT leak target host details.
  - [ ] T2.5: Tests in `pkg/sandbox/network_test.go` using `httptest.NewServer` for an
        upstream "real" target:
        - HTTPS-style `CONNECT` to allowed host → 200 tunnel established
        - HTTPS-style `CONNECT` to forbidden host → 403 with reason
        - Plain HTTP GET to allowed host → upstream response forwarded
        - Plain HTTP GET to forbidden host → 403 with reason
        - Audit callback invoked with correct `(host, allowed, reason)` for each case

- [ ] T3: `step.NetworkPolicy` field + Validate
  - [ ] T3.1: Add `NetworkPolicy *NetworkPolicy \`yaml:"network_policy,omitempty"\`` to `StepDef`
        in `internal/bench/experiment/design.go` — pointer type so absence is distinguishable
        from `{ allowed_domains: [], forbidden_domains: [] }`.
  - [ ] T3.2: Type alias to the package: `type NetworkPolicy = sandbox.NetworkPolicy` so YAML
        unmarshals via the public struct fields. Keep the actual struct in `pkg/sandbox` so
        `pkg/sandbox/network.go` is the single source of truth.
  - [ ] T3.3: Extend `ExperimentConfig.Validate()` to:
        - Reject unknown fields in `network_policy` (rely on `gopkg.in/yaml.v3` strict mode if
          available; otherwise validate after unmarshal by checking only known fields are
          populated).
        - Validate each domain pattern is non-empty and matches a basic shape (no spaces, no
          path components — host-only). Loose validation: reject patterns containing `/` or
          `?`.
  - [ ] T3.4: YAML round-trip test in `step_guardrails_test.go` (or a new
        `network_policy_test.go`): `network_policy: { forbidden_domains: ["*.internal"] }`
        unmarshals correctly, marshals back with `omitempty` honored.

- [ ] T4: Wire proxy lifecycle into the runtime (AC #4)
  - [ ] T4.1: In `internal/bench/experiment/runtime_runner.go`, inside `buildTaskDAG` per-step
        closure (where guardrails are set up around line 261), check `stepCopy.NetworkPolicy`.
        If non-nil with at least one of `AllowedDomains` or `ForbiddenDomains` populated:
        - Call `sandbox.NewProxyServer(policy, auditFn)` where `auditFn` calls into the
          existing `*SandboxFS.record(stepID, "network", host, allowed, reason)` so all
          node-boundary violations flow through one audit channel for story 10-6.
        - Defer `proxy.Close()` at the end of `runFn`.
        - Set `os.Setenv("HTTP_PROXY", proxy.URL())` and `os.Setenv("HTTPS_PROXY", proxy.URL())`
          before invoking the LLM, then `os.Unsetenv` (or restore prior values) on defer.
        - **CRITICAL:** `os.Setenv` is process-global. Concurrent steps would race. Use a
          per-step context-bound proxy URL injected into the LLM tool dispatcher's HTTP client
          configuration instead of process env vars when the LLM tool layer supports
          per-call clients (Phase 2 below). Phase 1 sets env vars but documents the
          parallelism limitation in `runtime_runner.go`.
  - [ ] T4.2: Document the limitation: until story 10-7+ explicitly enables parallel step
        execution, the env-var-based proxy injection is correct. When concurrency is added,
        switch to a per-request proxy via `http.Transport.Proxy` configured on the client
        used by the http-request tool.
  - [ ] T4.3: Audit recording: extend `AccessEntry.Action` to include `"network"` as a valid
        value (same shape as `"read"`/`"write"`/`"delete"`). The `Path` field carries the
        target host. The `Reason` field follows the same wording style as story 10-3:
        - AC1: `"domain matches forbidden pattern: <pattern>"`
        - AC2: `"domain not in allowed_domains"`

- [ ] T5: Tests — runtime integration
  - [ ] T5.1: `TestStepNetworkPolicy_ForbiddenDomain_Blocked` — step with
        `ForbiddenDomains: ["api.stripe.com"]`. Use a Go HTTP client configured with
        `http.Transport.Proxy = http.ProxyURL(proxy.URL())` to make a request to
        `api.stripe.com`. Expect 403 + audit entry with `Allowed: false` and matching reason.
  - [ ] T5.2: `TestStepNetworkPolicy_AllowedDomain_Forwarded` — step with
        `AllowedDomains: ["api.github.com"]` and an `httptest.NewServer` standing in as the
        target (rewrite the URL to point at the test server while still using the proxy).
        Verify the response body matches what the test server returned and audit shows
        `Allowed: true`.
  - [ ] T5.3: `TestStepNetworkPolicy_GlobForbidden_MatchesSubdomains` — `*.internal` blocks
        `api.internal` AND `db.cluster.internal`. Two requests, both blocked, audit has two
        entries with the matched pattern in each reason.
  - [ ] T5.4: `TestStepNetworkPolicy_NoPolicy_NoProxyInjected` — step with `NetworkPolicy: nil`.
        Verify `HTTP_PROXY` env var is NOT set during runFn execution. Assert via reading
        the env var inside the test's mock provider invocation.
  - [ ] T5.5: `TestStepNetworkPolicy_DenyPrecedence` — `AllowedDomains: ["api.example.com"]`
        AND `ForbiddenDomains: ["api.example.com"]`. Request to `api.example.com` is blocked.
  - [ ] T5.6: `TestStepNetworkPolicy_ProxyShutdownAfterStep` — proxy listener is closed after
        runFn returns. Verify by attempting to connect to the saved port AFTER step end and
        expecting connection refused.
  - [ ] T5.7: `TestValidate_RejectsUnknownNetworkPolicyField` — TaskDefinition with
        `network_policy: { allow_globs: [...] }` (typo). Validate returns an error mentioning
        the unknown field and the valid keys.
  - [ ] T5.8: `TestValidate_RejectsBadDomainPattern` — pattern containing `/` or `?` rejected.

- [ ] T6: Run `go test ./... -count=1` and `go test -race ./internal/bench/experiment/... ./pkg/sandbox/...`
       — no regressions, race-clean.

- [ ] T7: Documentation
  - [ ] T7.1: Add a precedence summary at the top of `pkg/sandbox/network.go`:
        `(1) ForbiddenDomains (deny wins)` → `(2) AllowedDomains (allow if listed)` →
        `(3) default (allow when no AllowedDomains configured)`.
  - [ ] T7.2: Document the env-var injection limitation under concurrent step execution
        (single-paragraph block in `runtime_runner.go` near the proxy lifecycle code).

## Dev Notes

### Architecture

**Three-layer enforcement model** (parallels stories 10-3 and 10-4):

```
1. step.NetworkPolicy (this story)        — domain allow/forbid list
2. proxy server intercepts HTTP            — prevents bypass of the policy
3. audit trail records violations         — feeds story 10-6 telemetry
```

**Why a proxy and not just a tool wrapper?** The architectural contract is: any HTTP traffic
from the step's environment is routed through the policy. If we wrapped only an
`http_request` tool, an LLM that emits Go code (or, in a future tool that spawns a
subprocess) could call `net/http` directly and bypass the check. The proxy is the network
boundary the OS provides.

**Why `HTTP_PROXY` / `HTTPS_PROXY` env vars?** They are the de-facto standard. Most HTTP
clients (Go's `http.DefaultTransport`, curl, requests, fetch in node, etc.) honor them
when configured via `http.ProxyFromEnvironment`. This makes the proxy transparent without
requiring tool-specific wiring.

**Ephemeral port?** TCP `127.0.0.1:0` lets the kernel assign a free port — no port-pool
management, no test flakiness from "port already in use". The test harness gets the URL
from `proxy.URL()` after `Listen` returns.

**Concurrency limitation (T4.1 critical note):** `os.Setenv` is process-global. If the
runtime ever executes two steps concurrently (parallel DAG layers), the env var
injection races. Phase 1 of this story uses env vars and documents the limitation;
Phase 2 (likely 10-7 or beyond when parallelism is enabled) switches to a per-request
proxy URL plumbed through the HTTP tool's client configuration. Since the current
runtime executes steps sequentially within a task, Phase 1 is sound for now.

### Glob semantics for hosts vs paths

`pkg/sandbox.GlobMatch` is path-glob (`/` as separator). Host glob is dot-separator.
A pattern `*.internal` should:
- Match `api.internal` (one label before)
- Match `a.b.internal` (multiple labels before)
- NOT match `internal` (zero labels before — bare domain)

The simplest implementation: split host on `.`, check the suffix labels against the
pattern's suffix labels, require at least one extra label before the wildcard.

```go
func hostMatch(pattern, host string) bool {
    if pattern == host {
        return true
    }
    if !strings.HasPrefix(pattern, "*.") {
        return false
    }
    suffix := pattern[1:] // includes the leading "."
    return len(host) > len(suffix) && strings.HasSuffix(host, suffix)
}
```

Adapt to lowercase if hosts are case-sensitive (DNS hosts are case-insensitive — normalize
both sides via `strings.ToLower` once).

### Backward compatibility (NFR40)

- Steps with no `network_policy` → no proxy started, no env vars set, no audit entries.
  Identical pre-story behavior.
- Empty `AllowedDomains` AND empty `ForbiddenDomains` → `CheckHost` returns `(true, "")`
  unconditionally. Effectively "policy declared but inert"; documented as such.
- Workspace-level proxy (if a future story adds one) would compose with this — but
  no workspace-level network policy exists today, so no precedence question to resolve
  this story.

### Files likely created or modified

- `pkg/sandbox/network.go` (new) — `NetworkPolicy`, `CheckHost`, `hostMatch`, `ProxyServer`
- `pkg/sandbox/network_test.go` (new) — unit tests for matcher + proxy
- `internal/bench/experiment/design.go` — `StepDef.NetworkPolicy` field + Validate
- `internal/bench/experiment/runtime_runner.go` — proxy lifecycle in per-step closure
- `internal/bench/experiment/runtime_runner_test.go` (or `step_guardrails_test.go`) —
  integration tests
- `internal/bench/experiment/sandbox.go` — extend `AccessEntry.Action` valid set to include
  `"network"` (currently a free-form string, but the test suite pins values)

### What NOT to do

- **Don't add an `http_request` tool to the experiment harness in this story.** This story
  builds the enforcement layer; tool integration is future work. The tests use
  Go's `http.Client` directly to validate the proxy, which is sufficient.
- **Don't implement TLS interception (MITM).** The proxy handles `CONNECT` via TCP
  forwarding only — it knows the host but cannot inspect the TLS payload. That's
  intentional: the contract is domain-level, not content-level. A content-level contract
  would require a CA cert installed in the LLM environment, which is out of scope and
  arguably anti-pattern for an experiment harness.
- **Don't make the proxy a singleton across the experiment.** Each step that declares a
  policy gets its own proxy. Different steps have different policies; sharing a proxy
  would couple their decisions and complicate audit attribution.
- **Don't pre-compile glob patterns.** Like file globs, host globs are checked at every
  request. Compile-once optimisation can come later if profiling shows it; the request
  rate from a single LLM call is too low to justify the complexity now.
- **Don't add concurrent-step support in this story.** Document the env var limitation
  and move on. Story 10-7+ owns DAG parallel execution.

### Previous story intelligence

- **10-3 introduces `step.FileAccess`** with the same audit-trail recording pattern.
  Use the SAME `sandbox.record(stepID, action, path, allowed, reason)` call shape so
  story 10-6's tally code can iterate one audit log uniformly. Action values become
  `{"read", "write", "delete", "network"}`.
- **10-4 introduces inline produces validation.** The audit recording happens at the
  same boundary (sandbox-level `record(...)`). Reusing the channel keeps things
  consistent.
- **10-0 added the defensive guardrail copy + Validate empty/duplicate StepID.** Apply
  the same defensive principles here: validate `network_policy` at task load, never
  alias slices in helpers, and fail-closed on policy parse errors.
- **10-2 fixed `template_file` resolution.** Not directly related to this story, but
  if any future hardening of the proxy reads a config file via `template_file`-like
  resolution, leverage the existing `task.TaskDir` plumbing.

### Telemetry hook (story 10-6)

This story's audit entries with `Action: "network"` and `Allowed: false` are the data
source for "network policy violations per node" in story 10-6. Stable Reason wording
(`"domain matches forbidden pattern: <pattern>"`, `"domain not in allowed_domains"`)
is a precondition for that story's parser. Treat the wording as part of the contract.

### Project context reference

- **Sprint status:** `_bmad-output/implementation-artifacts/sprint-status.yaml` —
  `10-5-http-proxy-network-sandbox: ready-for-dev` after this file lands.
- **Epic 10 plan:** `_bmad-output/planning-artifacts/epics.md` lines 371-397 (Story 7-5
  in the original numbering).
- **Recent commits:**
  - `a3296d9 feat(10-0): epic 9 debt cleanup` — landed sandbox TOCTOU fix, defensive
    guardrail copy, Validate StepID enforcement, taskContextContent step-level rendering,
    Compare TaskID alignment with three-way intersection in `CompareThree`
  - `46d041a feat(epic9): close epic 9 (DX) + retro + deferred-work triage and quick wins`
- **`pkg/sandbox` infrastructure already present** — `FileAccessPolicy`,
  `FileAccessMiddleware`, `GlobMatch`, `policy.go`. The new `NetworkPolicy` lives next
  to `FileAccessPolicy` for symmetry; the proxy server is the only genuinely new
  primitive in this story.

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

- T1+T2: `pkg/sandbox/network.go` — NetworkPolicy, CheckHost, hostMatch, ProxyServer (52 tests, race-clean)
- T3: `StepDef.NetworkPolicy *sandbox.NetworkPolicy` + validateNetworkPolicy + Validate() extension
- T4: Proxy lifecycle in LLM step runFn — start proxy, save/restore HTTP_PROXY/HTTPS_PROXY, defer close, audit via sandbox.record
- T5: `internal/bench/experiment/network_policy_test.go` — YAML round-trip, Validate rejections, env var set/unset/restore
- T6: `go test ./... -count=1` passes (2 pre-existing unrelated failures); `go test -race ./internal/bench/experiment/... ./pkg/sandbox/...` — 246 tests race-clean

### File List

- `pkg/sandbox/network.go` (new)
- `pkg/sandbox/network_test.go` (new)
- `internal/bench/experiment/design.go` (modified)
- `internal/bench/experiment/runtime_runner.go` (modified)
- `internal/bench/experiment/network_policy_test.go` (new)

### Review Findings

- [x] [Review][Decision] AC6: yaml.v3 silently ignores unknown fields in `network_policy` — `validateNetworkPolicy` only rejects bad patterns, not unknown keys like `allow_globs`. Fixing requires `yaml.Decoder.KnownFields(true)` in `LoadTasks`, which is an architectural decision affecting all YAML parsing. — Choose: (a) strict KnownFields for network_policy block only, (b) strict KnownFields globally, (c) defer AC6 with a doc note.
- [x] [Review][Patch] Proxy self-loop: `handleHTTP` uses `http.DefaultTransport` which reads `HTTP_PROXY` env var → routes back to itself [`pkg/sandbox/network.go:179`]
- [x] [Review][Patch] `WriteHeader(200)` committed before Hijacker type assertion in `handleCONNECT` — client gets 200+502 body on hijack failure [`pkg/sandbox/network.go:142-146`]
- [x] [Review][Patch] CONNECT tunnel: only `<-done` once for two goroutines — second goroutine races against defer closes [`pkg/sandbox/network.go:156-159`]
- [x] [Review][Patch] `bufio.ReadWriter` from `hijacker.Hijack()` discarded — pipelined bytes lost before io.Copy [`pkg/sandbox/network.go:150`]
- [x] [Review][Patch] `Close()` closes listener only, not `ps.server` — in-flight CONNECT goroutines and http.Server leak after step ends [`pkg/sandbox/network.go:106-108`]
- [x] [Review][Patch] `NO_PROXY`/`no_proxy` not saved/restored — existing env var bypasses proxy enforcement for listed hosts [`internal/bench/experiment/runtime_runner.go:492-508`]
- [x] [Review][Patch] Empty host not rejected in `handleHTTP` — `r.URL.Hostname()` returns `""` for malformed requests, passes `CheckHost("")` on open allow-list [`pkg/sandbox/network.go:163`]
- [x] [Review][Patch] `SplitHostPort` failure falls back to `r.Host` with port in `handleCONNECT` — `"api.example.com:443"` fails to match pattern `"api.example.com"` [`pkg/sandbox/network.go:120-123`]
- [x] [Review][Patch] `validateNetworkPolicy` accepts `"*.*.example.com"` — double wildcard passes validation but never matches in `hostMatch` [`internal/bench/experiment/design.go:450-468`]
- [x] [Review][Patch] `validateNetworkPolicy` accepts `"api.example.com:8080"` — port suffix passes validation but never matches bare hostname [`internal/bench/experiment/design.go:450-468`]
- [x] [Review][Patch] `validateNetworkPolicy` should reject bare `"*"` — silently denies everything; confusing for authors who expect "allow all" [`pkg/sandbox/network.go`]
- [x] [Review][Patch] T4.3 untested: audit wiring (`Action="network"`, `Path=host`) never asserted in integration tests [`internal/bench/experiment/network_policy_test.go`]
- [x] [Review][Patch] Non-discriminating test for `'?'` in pattern validation — error message always includes both chars; dropping `?` detection wouldn't fail the test [`internal/bench/experiment/network_policy_test.go`]
- [x] [Review][Patch] Tests use `os.Setenv`/`os.Unsetenv` instead of `t.Setenv` — unsafe for future `t.Parallel()` [`internal/bench/experiment/network_policy_test.go:153,192,228`]
- [x] [Review][Defer] `os.Setenv("HTTP_PROXY")` process-global — documented Phase 1 limitation, deferred to 10-7 when parallel DAG layers enabled [`runtime_runner.go:479-481`] — deferred, documented in code
- [x] [Review][Defer] Network audit reason may inflate file `ScopeCreepRate` metric — verify `ComputeActionMetrics` does not match network deny reasons as scope creep — deferred, needs investigation

## Change Log

| Date | Change | Notes |
|---|---|---|
| 2026-04-28 | Story 10-5 created from Epic 10 plan (Story 7-5 in epics.md) | Builds on `pkg/sandbox` (existing FileAccessPolicy + GlobMatch + middleware pattern) and the audit-trail conventions established by stories 10-3 and 10-4. Phase 1 uses env-var-based proxy injection (single-step at a time); Phase 2 (per-request proxy under concurrent execution) deferred to story 10-7+. |
