# Story 2.4: Registry Notification and Update

Status: ready-for-dev

## Story

As a **developer**,
I want doctor to notify me when a newer research registry is available and let me update it,
so that my recommendations stay current with the latest research.

## Acceptance Criteria

1. When the embedded registry version is older than the latest published version, doctor displays a notification: "Research registry update available. Run `forgent doctor --update-registry`" (FR17)
2. Notification displays only once per run, after all findings, in both TTY and plain mode
3. `forgent doctor --update-registry` downloads the latest registry to `.forgent/research-registry.yaml` with checksum verification (NFR16), displaying the target URL before download begins (NFR22)
4. The download URL is configured in the embedded registry YAML (`update_url` field) — not hardcoded in Go
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Registry version notification (AC: #1, #2)
  - [ ] 1.1: Add `LatestVersion` field to registry YAML format — the embedded registry knows the latest published version number
  - [ ] 1.2: In `internal/doctor/registry/registry.go` (from Story 2.1), add `NeedsUpdate() bool` method comparing active version against `LatestVersion`
  - [ ] 1.3: In `internal/cmd/doctor.go`, after analysis, check `registry.NeedsUpdate()` and print notification if true
  - [ ] 1.4: Notification displays only once per run, after all findings, in both TTY and plain mode
  - [ ] 1.5: Write tests for: version match (no notification), version mismatch (notification shown), no LatestVersion field (no notification)

- [ ] Task 2: Registry update command (AC: #3, #4)
  - [ ] 2.1: Add `--update-registry` flag to doctor command
  - [ ] 2.2: Create `internal/doctor/registry/updater.go` — `Update(targetDir string) error`
  - [ ] 2.3: Update flow: (a) display target URL, (b) download registry YAML, (c) verify checksum (SHA-256 from a companion `.sha256` file or header), (d) write to `.forgent/research-registry.yaml` via `os.MkdirAll` + `os.WriteFile`
  - [ ] 2.4: The download URL is configured in the embedded registry YAML (`update_url` field) — not hardcoded in Go
  - [ ] 2.5: Show the URL before downloading (NFR22: network request URLs visible before sending)
  - [ ] 2.6: Handle errors: network failure, checksum mismatch, write permission. All errors are actionable.
  - [ ] 2.7: Write tests with an httptest server serving a mock registry + checksum

- [ ] Task 3: CLI integration (AC: #3)
  - [ ] 3.1: If `--update-registry` is passed, run update and exit (no analysis)
  - [ ] 3.2: Add `--update-registry` flag help text with example
  - [ ] 3.3: Write smoke test for the flag

## Dev Notes

### Architecture References

- **ADR-3 (Research Registry):** "Growth phase adds `forgent doctor --update-registry` to download newer versions."
- **ADR-7 (Configuration):** `--update-registry` is a CLI flag, not a config option. The download URL lives in the registry YAML itself.
- **NFR16:** Registry checksum verification.
- **NFR18:** Local-first — `--update-registry` is the ONLY network operation in doctor. URL is shown before download.
- **NFR22:** Network request URLs visible before sending.

### Dependencies on Previous Stories

- **Story 2.1 (Research Registry):** Must be complete. This story adds `NeedsUpdate()`, `LatestVersion`, `update_url` to the registry package.

### Registry Update Flow

```
1. Read update_url from active registry
2. Print: "Downloading registry from <url>..."
3. GET <url> → response body
4. GET <url>.sha256 → expected checksum
5. Compute SHA-256 of response body
6. Compare checksums — if mismatch: "Checksum verification failed. Registry not updated."
7. Write to .forgent/research-registry.yaml
8. Print: "Registry updated to version <version>"
```

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done" — 3-layer adversarial code review is non-negotiable.
- **Action #5:** "No undocumented scope creep."

### What NOT to Build

- Do NOT implement auto-update (checking on every run and downloading automatically) — only check version, never auto-download. `--update-registry` is explicit.
- Do NOT implement registry diff/changelog display — just download the new version.
- Do NOT implement rollback for registry updates — local override can be manually deleted to revert to embedded.

### Project Structure

New files:
- `internal/doctor/registry/updater.go` — Update() function with download + checksum
- `internal/doctor/registry/updater_test.go` — tests with httptest server

Modified files:
- `internal/cmd/doctor.go` — `--update-registry` flag, version notification after analysis
- `internal/doctor/registry/registry.go` (from Story 2.1) — add NeedsUpdate(), LatestVersion field
- `internal/doctor/registry/data/research-registry.yaml` (from Story 2.1) — add latest_version, update_url fields

## Testing Strategy

- **Version notification tests:** Mock registry with matching/mismatching versions. Verify notification text and conditions. No LatestVersion field → no notification.
- **Update tests:** `httptest.NewServer` serving mock YAML + checksum. Test: success, checksum mismatch, network error, write permission error.
- **CLI flag test:** `--update-registry` runs update and exits without analysis.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-3] — Research registry hosting, update mechanism
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-7] — Configuration resolution precedence
- [Source: _bmad-output/planning-artifacts/prd.md#FR17] — Registry version notification
- [Source: _bmad-output/planning-artifacts/prd.md#FR18] — Registry update command
- [Source: _bmad-output/planning-artifacts/prd.md#NFR16] — Registry checksum verification
- [Source: _bmad-output/planning-artifacts/prd.md#NFR22] — Network request URLs visible before sending

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
