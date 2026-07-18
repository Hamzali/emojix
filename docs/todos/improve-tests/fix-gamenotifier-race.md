# T01 — Fix GameNotifier production data race

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** T02, T04, T11, T12, T18

## Problem

`service/gamenotifier.go` stores `subs []gameSub` with **no synchronization**. Every method
mutates or iterates it:

- `Sub` appends to `gn.subs`.
- `Sub`'s returned `cleanup` does `slices.DeleteFunc` on `gn.subs`.
- `Subs`, `Pub`, `PubAll` iterate `gn.subs`.

Concurrent SSE connections (the normal case — multiple users in one game, plus
`go e.gameNotifier.Pub(...)` goroutines) therefore race on the slice. Confirmed by
`go test -race ./service/...` failing today.

This is a **production correctness bug**, not a test artifact.

## Goal

Make `gameNotifier` safe for concurrent use with minimal, dependency-free changes.

## Approach

1. Add a `sync.RWMutex` to `gameNotifier`.
2. `Sub`: take write lock, append, release; return a `cleanup` that takes the write lock
   before `DeleteFunc`.
3. `Subs`: take read lock while iterating and copying.
4. `Pub` / `PubAll`: take read lock while iterating. **Do not** hold the lock while sending
   on `s.NotifChan` — collect the target subs under the lock, then send outside the lock
   (existing sends are unbuffered/blocking; holding the mutex across a blocking send would
   deadlock with a concurrent `cleanup`).
5. Keep the public `GameNotifier` interface unchanged.

## Out of scope

- Channel cleanup strategy (`TODOs.md` already tracks "cleanup/close realtime channels").
- Changing `Pub`/`PubAll` to be non-blocking (separate backlog item).

## Acceptance criteria

- [ ] `go test -race ./service/...` reports no race in `TestGameNotifierPubSub` /
      `TestGameNotifierSubs`.
- [ ] All existing notifier tests still pass without modification of their assertions.
- [ ] No new public API; `NewGameNotifier()` signature unchanged.
- [ ] `go vet ./service/...` clean.

## Files touched

- `service/gamenotifier.go`

## Notes / risks

- Sending outside the lock means a sub's channel could be closed between unlock and send.
  Today channels are never closed by `gameNotifier` (only `cleanup` removes the sub), so
  this is safe. Double-check no test closes the returned channel while a `Pub` is in flight.
- Keep the existing `time.Now()` `LastMsgAt` field even though it is currently unused, to
  avoid unrelated diffs.