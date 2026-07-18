# T04 — Extend MockGameNotifier

- **Status:** `[ ]`
- **Depends on:** T01 (lock the real notifier first; the mock's concurrency story follows)
- **Unblocks:** T08, T11, T12 (`PubAll` path is exercised by `onTurnEnd`)

## Problem

`service/gamenotifier_mock.go` `MockGameNotifier`:
- has no `PubAllMock`; `PubAll` falls through to the embedded **nil** `GameNotifier`
  interface → panic. `onTurnEnd` calls `PubAll`, so the turn-end path is currently
  untestable.
- sets `PubCalled = true` from inside whatever goroutine calls `Pub` (production spawns
  `go e.gameNotifier.Pub(...)`), which races with the test reading `PubCalled` (T02).

## Goal

Make `MockGameNotifier` cover the full `GameNotifier` interface and be safe to assert
against under `-race`.

## Approach

1. Add `PubAllMock func(gameID string, notif GameNotification)` + `PubAllCalled bool` to
   `MockGameNotifier`, implementing `PubAll`.
2. Protect `PubCalled`/`PubAllCalled` writes and reads. Default to the simplest correct
   option: use a `sync.Mutex` (or `atomic.Bool`) for the `*Called` flags, and document that
   tests should additionally wait for completion via channels (T02 pattern) before asserting.
3. Keep the embedded `GameNotifier` interface so unset methods still panic (deliberate).
4. Do **not** change the real `GameNotifier` (that's T01).

Default the mock fields defensively so constructing `MockGameNotifier{}` without a
`PubAllMock` doesn't nil-deref on `PubAll` — return a no-op instead, since
`Pub`/`PubAll` legitimately "do nothing" in many negative test cases. (Contrast: keep
`Sub`/`Subs` panicking if unset — those are required to return values, so a panic is the
correct "you forgot to wire it" signal.)


## Out of scope

- Rewriting the real notifier (T01).
- Reformulating notification data (backlog).

## Acceptance criteria

- [ ] `PubAll` is exercisable via `PubAllMock` and updates `PubAllCalled`.
- [ ] `go test -race ./usecase/...` shows no race attributable to `PubCalled`/`PubAllCalled`.
- [ ] Existing `MockGameNotifier` consumers (locator: grep `MockGameNotifier`) compile and
      pass without behavioral change.
- [ ] `go vet ./service/... ./usecase/...` clean.

## Files touched

- `service/gamenotifier_mock.go`

## Notes

- The "default no-op for Pub/PubAll, panic for Sub/Subs if unset" rule should be applied
  consistently to the whole mock file; call it out in a comment so future mock authors
  follow it.
- If T02 lands nearby, coordinate on whether the `*Called` flags become `atomic.Bool` in
  this task or T02 — pick one owner to avoid churn. Default: do the synchronization here,
  T02 just consumes it.