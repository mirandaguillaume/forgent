# Risk Score — Story 10-5: HTTP Proxy Network Sandbox

**Risk Score: 7 / 10**

---

## Scoring Breakdown

### Diff Scope and Complexity — 3 new files, 2 modified files, ~511 lines in key paths

| Dimension | Assessment |
|-----------|------------|
| New surface | `pkg/sandbox/network.go` — 193 lines, new package-level HTTP proxy server |
| Integration depth | `runtime_runner.go` — proxy lifecycle wired into every LLM step runFn closure; shares process-global env vars |
| Validation layer | `design.go` — `validateNetworkPolicy` is a new validation path; gaps remain (port suffixes, bare `*`) |
| Concurrency exposure | `os.Setenv` is process-global; documented as sequential-only but not enforced |

Complexity is high: the proxy is a real HTTP/CONNECT proxy with goroutine tunnelling, hijacking, and bidirectional io.Copy. Errors compound across layers (env var → Go transport → proxy handler → audit trail).

---

### Test Coverage

| File | Coverage |
|------|----------|
| `pkg/sandbox/network.go` | 90.4% overall — `handleCONNECT` 80.6%, `handleHTTP` 89.5%, `NewProxyServer` 85.7% |
| `internal/bench/experiment/design.go::validateNetworkPolicy` | 90.9% |
| `internal/bench/experiment/runtime_runner.go::buildTaskDAG` | 75.9% — proxy lifecycle branch partially covered |

Coverage numbers look acceptable on the surface, but the uncovered branches in `handleCONNECT` are exactly the error paths where bugs C2 (WriteHeader ordering) and C5 (SplitHostPort fallback) live. The integration test `TestNetworkPolicy_AuditEntry_RecordedOnProxy` explicitly acknowledges it does not make real HTTP requests through the proxy, leaving the proxy's actual enforcement path untested end-to-end.

---

### Lint / Static Analysis

- `go vet`: no issues
- `go build`: clean
- Race detector (`-race`): clean under current sequential test execution
- `//nolint:errcheck` appears 5 times in `network.go` and `runtime_runner.go` — two of those (C7) suppress errors that would leave the system in a silently broken state

---

### Confirmed Bugs by Severity

| ID | Severity | Description |
|----|----------|-------------|
| C8 | CRITICAL | `proxy.Close()` only closes listener; in-flight tunnels and goroutines outlive the step |
| C3 | CRITICAL | `http.DefaultTransport` pools connections across steps, bypassing per-step audit |
| C1 | HIGH | CONNECT tunnel goroutine leak — only one of two copy goroutines awaited |
| C6 | HIGH | `NO_PROXY` not saved/restored; forbidden hosts can bypass enforcement |
| C2 | HIGH | `WriteHeader(200)` before Hijack assertion — protocol violation on HTTP/2 or non-hijackable writers |
| C7 | HIGH | `os.Setenv` error discarded; proxy started but traffic bypasses it silently |
| C5 | MEDIUM | `SplitHostPort` failure falls back to raw `r.Host` with port in policy check |
| C4 | MEDIUM | Empty host passes `CheckHost` on malformed proxy requests |
| C12 | MEDIUM | Network audit reason leaks into `OutOfScopeAttempts` metric |
| C10 | MEDIUM | Port suffixes in domain patterns silently never match |
| C9 | LOW | Tests use `os.Setenv` instead of `t.Setenv` — race-unsafe for future parallelisation |
| C11 | LOW | Bare `"*"` pattern silently denies everything instead of matching everything |

2 CRITICAL + 4 HIGH bugs confirmed. The two critical bugs mean the proxy's core guarantee — "traffic is audited and controlled for the duration of the step" — does not hold reliably even in the sequential case.

---

### Risk Rationale

**Score 7/10.** The story introduces a net-new security enforcement layer (network sandboxing) whose correctness is the entire value proposition. Two critical bugs directly undermine that guarantee: connections can escape to a pooled transport that bypasses the policy proxy, and the proxy server's goroutines and connection state survive past `Close()`. Neither defect would be caught by the current test suite because the integration tests mock the LLM provider and do not make real HTTP calls through the proxy. The remaining HIGH bugs compound the risk in production environments where `NO_PROXY` is set or where `os.Setenv` may fail. Correctness must be verified against the actual Go HTTP stack before this ships as a security primitive.
