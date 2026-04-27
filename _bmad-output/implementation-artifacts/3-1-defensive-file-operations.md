# Story 3.1: Defensive File Operations

Status: ready-for-dev

## Story

As a **developer**,
I want automatic backups and safe writes for any file doctor modifies,
so that I can always recover if something goes wrong.

## Acceptance Criteria

1. `internal/fileops/` package with `SafeWrite(path string, content []byte) error` — creates backup at `path.bak`, writes to temp file, atomically renames to target
2. If any step of SafeWrite fails, the original file is preserved
3. `Restore(path string) error` restores the original from `.bak` file
4. File locking prevents concurrent `--fix` runs from corrupting the same file
5. Ctrl+C during `--fix` produces no partial write — backup remains intact, clean exit
6. Backups older than 7 days (configurable) are cleaned up when doctor runs
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Core SafeWrite and Restore (AC: #1, #2, #3)
  - [ ] 1.1: Create `internal/fileops/safe_write.go` — `SafeWrite(path string, content []byte) error`
  - [ ] 1.2: SafeWrite flow: (a) copy original to `path.bak`, (b) write content to `path.tmp`, (c) `os.Rename(path.tmp, path)` for atomic replace
  - [ ] 1.3: If step (a) fails, return error immediately (original untouched)
  - [ ] 1.4: If step (b) fails, remove `path.tmp` if it exists, return error (original + backup intact)
  - [ ] 1.5: If step (c) fails, `path.tmp` remains but original is intact — return error with "manual rename may be needed"
  - [ ] 1.6: Create `Restore(path string) error` — `os.Rename(path+".bak", path)`
  - [ ] 1.7: Write tests for each failure scenario using `t.TempDir()`

- [ ] Task 2: File locking (AC: #4)
  - [ ] 2.1: Create `internal/fileops/lock.go` — `Lock(path string) (*FileLock, error)` and `(*FileLock).Unlock() error`
  - [ ] 2.2: Use advisory file locking with a `.lock` file: create `path.lock`, write PID, defer removal
  - [ ] 2.3: If `.lock` exists and process is still alive, return actionable error: "File is locked by another forgent process (PID: X). Wait or remove <path>.lock"
  - [ ] 2.4: If `.lock` exists but process is dead (stale lock), remove and acquire
  - [ ] 2.5: Integrate locking into SafeWrite — acquire lock before backup, release after rename
  - [ ] 2.6: Write tests: concurrent goroutines, stale lock cleanup, lock contention

- [ ] Task 3: Signal handling (AC: #5)
  - [ ] 3.1: Create `internal/fileops/signal.go` — `WithCleanup(ctx context.Context) (context.Context, context.CancelFunc)`
  - [ ] 3.2: Trap SIGINT/SIGTERM, cancel context, ensure cleanup runs before exit
  - [ ] 3.3: SafeWrite checks context before each step — if cancelled, abort without partial write
  - [ ] 3.4: Write tests simulating signal during each SafeWrite phase

- [ ] Task 4: Backup cleanup (AC: #6)
  - [ ] 4.1: Create `internal/fileops/cleanup.go` — `CleanupBackups(dir string, maxAge time.Duration) (int, error)`
  - [ ] 4.2: Walk directory for `.bak` files, remove those older than `maxAge`
  - [ ] 4.3: Default 7 days, configurable via `--backup-retention` flag (future) or programmatic API
  - [ ] 4.4: Return count of cleaned files for debug logging
  - [ ] 4.5: Write tests with temp dir containing old and new `.bak` files

## Dev Notes

### Architecture References

- **ADR-2 (Quality Gate):** "Fail-closed: any validation failure → reject rewrite, keep original, show error." SafeWrite is the enforcement mechanism — if quality gate rejects, SafeWrite is never called.
- **Architecture enforcement rule #5:** "Use `fileops.SafeWrite()` for any user file modification." This package is the single write path for all user files.
- **Architecture anti-pattern:** "Don't write to user files without `fileops.SafeWrite()`."
- **NFR10:** "Backup is created BEFORE any file modification."
- **NFR15:** "Concurrent doctor runs on the same file do not corrupt it."
- **NFR17:** "Doctor backups are cleaned up after 7 days by default."
- **NFR34:** "Ctrl+C clean exit — no orphan processes, no partial writes."

### Atomic Rename Caveat

`os.Rename()` is atomic on the same filesystem (POSIX guarantee). The temp file must be in the same directory as the target to ensure same-filesystem. Use `filepath.Dir(path)` for temp file location.

On Windows, `os.Rename()` may fail if the target exists. Use `os.Remove(path)` + `os.Rename(tmp, path)` as fallback, accepting a small non-atomic window. Document this limitation.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done — 3-layer review is non-negotiable."
- **Recurring finding:** "Tests dependent on relative paths" — use `t.TempDir()` for all file-system tests.
- **Recurring finding:** "CRLF not handled" — SafeWrite must preserve exact byte content, no line-ending transformation.

### What NOT to Build

- Do NOT implement the quality gate validation — that's Story 3.2
- Do NOT implement the rewrite logic — that's Story 3.3
- Do NOT implement cross-platform file locking (flock/LockFileEx) — advisory `.lock` files are sufficient for MVP
- Do NOT integrate with the doctor CLI yet — Story 3.3 will wire SafeWrite into the `--fix` flow

### Project Structure

New files:
- `internal/fileops/safe_write.go` — SafeWrite, Restore
- `internal/fileops/safe_write_test.go`
- `internal/fileops/lock.go` — FileLock, Lock, Unlock
- `internal/fileops/lock_test.go`
- `internal/fileops/signal.go` — WithCleanup, signal trapping
- `internal/fileops/signal_test.go`
- `internal/fileops/cleanup.go` — CleanupBackups
- `internal/fileops/cleanup_test.go`

## Testing Strategy

- **SafeWrite tests:** Table-driven for success, backup failure, write failure, rename failure. Verify original file preserved in each failure case. Use `t.TempDir()`.
- **Restore tests:** Restore from backup, no backup exists (error), backup is corrupted.
- **Lock tests:** Concurrent goroutines attempting SafeWrite on same file. Stale lock detection. Lock contention error message.
- **Signal tests:** Use `syscall.Kill(os.Getpid(), syscall.SIGINT)` in test to simulate Ctrl+C. Verify no partial write.
- **Cleanup tests:** Temp dir with `.bak` files of varying ages. Verify only old ones removed.
- **Race test:** `go test -race` on concurrent SafeWrite scenarios.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-2] — Quality gate fail-closed validation
- [Source: _bmad-output/planning-artifacts/architecture.md#Defensive-File-Operations-Pattern] — SafeWrite code pattern
- [Source: _bmad-output/planning-artifacts/prd.md#FR19] — Read-only default
- [Source: _bmad-output/planning-artifacts/prd.md#FR22] — Automatic backup
- [Source: _bmad-output/planning-artifacts/prd.md#FR23] — Restore from backup
- [Source: _bmad-output/planning-artifacts/prd.md#NFR10] — Backup before modification
- [Source: _bmad-output/planning-artifacts/prd.md#NFR15] — Concurrent file safety
- [Source: _bmad-output/planning-artifacts/prd.md#NFR17] — Backup cleanup
- [Source: _bmad-output/planning-artifacts/prd.md#NFR34] — Clean exit on Ctrl+C
- [Source: _bmad-output/implementation-artifacts/epic-1-retro-2026-03-30.md] — Retro lessons

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
