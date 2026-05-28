# Review Comments — Story 10-5: HTTP Proxy Network Sandbox

## C1 — CONNECT tunnel goroutine leak: only one of two copy goroutines is awaited
**File:** pkg/sandbox/network.go:156-159
Code: `<-done` (single receive from a channel fed by two goroutines)
Only one receive is issued. When the first half-connection closes the handler returns, defers fire and close both connections, but the second io.Copy goroutine remains alive referencing closed file descriptors.
Fix: add a second `<-done` or use sync.WaitGroup.

## C2 — WriteHeader(200) sent before the Hijack type-assertion
**File:** pkg/sandbox/network.go:142-146
Code: `w.WriteHeader(http.StatusOK)` then `hijacker, ok := w.(http.Hijacker); if !ok { http.Error(w, "502...", ...) }`
If w does not implement Hijacker the 200 is already flushed; the subsequent http.Error call writes a 502 body but the status line is 200. On HTTP/2 the Hijack path is never available so this is a guaranteed protocol violation.
Fix: assert Hijacker first; only call WriteHeader(200) after the assertion succeeds.

## C3 — http.DefaultTransport reused in proxy handleHTTP — connection pool bypasses per-step proxy
**File:** pkg/sandbox/network.go:179
Code: `resp, err := http.DefaultTransport.RoundTrip(outReq)`
DefaultTransport is package-global and pools connections. Connections dialed through one step's proxy remain pooled and are reused in the next step, bypassing that step's NetworkPolicy audit entirely. Additionally the proxy handler itself inherits HTTP_PROXY so it may loop back to itself.
Fix: create a dedicated http.Transport per ProxyServer with Proxy: nil.

## C4 — handleHTTP: empty host passes CheckHost when r.URL.Hostname() returns ""
**File:** pkg/sandbox/network.go:163
Code: `host := r.URL.Hostname()`
A relative-URL request (malformed proxy client) produces an empty host. CheckHost("") passes with an empty AllowedDomains list, allowing the request with no audit entry naming a host.
Fix: guard `if host == "" { http.Error(w, "400 Bad Request", 400); return }`.

## C5 — handleCONNECT: SplitHostPort failure falls back to raw r.Host (with port) for policy check
**File:** pkg/sandbox/network.go:120-123
Code: `host, _, err := net.SplitHostPort(r.Host); if err != nil { host = r.Host }`
If SplitHostPort fails, the raw value (possibly "api.example.com:" with trailing colon) is passed to CheckHost. Pattern "api.example.com" will not match "api.example.com:", silently denying a legitimate host. The audit logs the wrong host string. net.Dial("tcp", r.Host) on line 135 then fails, but only after policy was already evaluated.
Fix: return 400 on SplitHostPort failure instead of falling back.

## C6 — NO_PROXY / no_proxy not saved or restored
**File:** internal/bench/experiment/runtime_runner.go:492-508
Code: Only HTTP_PROXY and HTTPS_PROXY are saved/restored; NO_PROXY is untouched.
If the environment has NO_PROXY=api.example.com, Go's ProxyFromEnvironment bypasses the step's enforcement proxy for that host even if the NetworkPolicy forbids it. The audit trail shows zero entries for those bypassed requests.
Fix: save and clear NO_PROXY/no_proxy for the duration of the step, restore in defer.

## C7 — os.Setenv errors silently discarded; proxy runs but traffic bypasses it undetected
**File:** internal/bench/experiment/runtime_runner.go:494-495
Code: `os.Setenv("HTTP_PROXY", proxy.URL()) //nolint:errcheck`
If Setenv fails (ENOMEM, read-only env), the proxy is started and bound but the Go runtime uses the old HTTP_PROXY. LLM calls proceed with no policy enforcement and an empty audit trail.
Fix: check the error and return it, closing the proxy before returning.

## C8 — proxy.Close() only closes the listener, not the http.Server
**File:** pkg/sandbox/network.go:106-108
Code: `func (ps *ProxyServer) Close() error { return ps.listener.Close() }`
Closing the listener stops new accepts but does not terminate in-flight requests or their goroutines. Active CONNECT tunnels and their two io.Copy goroutines remain alive after Close() returns. The step defer returns and HTTP_PROXY is restored, but the old proxy goroutines continue running against the closed policy.
Fix: call ps.server.Close() instead, which closes the listener AND forces-closes all active connections.

## C9 — Tests use os.Setenv/os.Unsetenv instead of t.Setenv — env leaks under parallel execution
**File:** internal/bench/experiment/network_policy_test.go:153-154, 192-193, 228-229
Code: `os.Unsetenv("HTTP_PROXY")` / `os.Setenv("HTTP_PROXY", sentinel)` / `defer os.Unsetenv(...)`
t.Setenv is the Go 1.17+ idiom that registers cleanup via t.Cleanup, safe for future t.Parallel use. If tests are ever parallelised without fixing env handling, concurrent tests will race on HTTP_PROXY with no detector warning.
Fix: replace all raw os.Setenv/os.Unsetenv pairs in tests with t.Setenv.

## C10 — validateNetworkPolicy does not reject patterns with a port suffix
**File:** internal/bench/experiment/design.go:450-468
Code: Only "/" and "?" are rejected; "api.example.com:443" passes validation.
CheckHost receives a bare hostname (port already stripped). A pattern containing a port never matches, silently turning that allowlist entry into a dead rule with no error. No test covers this.
Fix: add `strings.ContainsAny(pat, ":")` check and return an error.

## C11 — hostMatch: bare "*" pattern silently never matches (WAI but surprising)
**File:** pkg/sandbox/network.go:56-71
Code: `if !strings.HasPrefix(pattern, "*.") { return false }` — bare "*" takes the false branch.
A task author writing `allowed_domains: ["*"]` expecting "allow all" gets "deny all" because the non-empty allowlist matches nothing. validateNetworkPolicy does not reject it and there is no test documenting this behaviour.
Fix: reject bare "*" in validateNetworkPolicy with an explicit error message pointing to the correct idiom (empty allowed_domains list).

## C12 — Network audit reason "domain not in allowed_domains" matches OutOfScopeAttempts counter
**File:** internal/bench/experiment/runtime_runner.go:485 + internal/bench/experiment/sandbox.go:223
Code: `stepSandbox.record(stepCopy.ID, "network", host, allowed, reason)` where reason = "domain not in allowed_domains"
ComputeActionMetrics uses `strings.Contains(e.Reason, "not in allowed")` to count OutOfScopeAttempts. The network denial reason contains that substring, so network policy denials inflate the file-scope-creep metric.
Fix: change the network denial reason to avoid the substring, e.g. "domain blocked: not listed in allowed_domains", or add a dedicated NetworkDenials counter.
