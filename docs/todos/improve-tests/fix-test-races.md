# T02 — Fix test-side data races

- **Status:** `[ ]`
- **Depends on:** T01 (same concurrency surface; land T01 first so notifications stay correct)
- **Unblocks:** T18

## Problem

`go test -race ./...` currently fails in tests, independent of the production bug in T01,
because tests read fields that goroutines write:

1. **`service/gameloop_test.go` `TestGameLoop_DoubleEndGameTurn`** — reads `callCount` from
   the test goroutine while the loop goroutine writes it (`callCount++`).
2. **`usecase/joingame_test.go` and `usecase/kickinactiveuser_test.go`** — production code
   calls `Pub` via `go e.gameNotifier.Pub(...)` (a goroutine), and the `MockGameNotifier`
   sets `m.PubCalled = true` inside that goroutine. Tests assert `PubCalled` (and
   `SetPlayerStateCalled`) on the main goroutine → race.

## Goal

Make all tests `-race` clean by synchronizing the assertions, **without** weakening the
tests' behavior checks.

## Approach

### For the game loop counter

- Replace the bare `callCount` int with a buffered `chan struct{}` (size 4) or guard it with
  a `sync.Mutex`. Prefer the channel pattern already used by the other game loop tests
  (`calls := make(chan string, 1)`) for consistency. Assert exactly one call via a
  `select { case <-calls: t.Fatal("called twice"); default: }` after the expected one.

### For the notifier mock flag races

Two parts:

1. **Mock safety (also part of T04):** protect `PubCalled`/`SetPlayerStateCalled` writes in
   mocks with synchronization. But note `PubCalled` lives on `MockGameNotifier`
   (service pkg) and `SetPlayerStateCalled` lives on `MockGameRepository` (repo pkg). Both
   are written from goroutines spawned by production code.
2. **Test synchronization:** the reliable fix is for tests to wait for the `go Pub(...)`
   goroutine to finish before asserting. The join tests already use a `pubCh` channel in
   some cases — extend that pattern to every case that currently asserts `PubCalled`/
   `SetPlayerStateCalled`. Use channels (block on `<-pubCh` with a `time.After` guard) or
   `sync.WaitGroup` set inside the mock's `PubMock`.

Concretely, in `TestJoinGame`:
- `"adds player"` already uses `pubCh` — keep.
- `"fails to add if player is already in game and active"` — `Pub` must NOT be called. Add a
  `PubMock` that sends on a channel only if called; the test should assert nothing arrives
  within a short timeout (e.g. `select { case <-pubCh: t.Fatal; case <-time.After(50ms): }`).
- `"reactivates user joined and kicked before"` — same as above.
- `"fails to add if room is full"` — same.

In `TestKickInactiveUser`:
- `"kicks inactive user"` already uses `pubCh` — keep.
- `"keeps user if its active"` — add the negative-timeout assert.

## Out of scope

- Removing `go e.gameNotifier.Pub(...)` from production (that's a separate design call,
  tracked in TODOs as "cleanup channels").
- Changes to the real `gameNotifier` — that's T01.

## Acceptance criteria

- [ ] `go test -race ./service/... ./usecase/...` green.
- [ ] No test sleeps longer than `100ms` for synchronization (use channels primarily).
- [ ] Negative assertions ("must NOT be called") use a bounded `select`/timeout, not
      sleeping then checking a flag.
- [ ] Existing test behavior (which methods get called) still asserted.

## Files touched

- `service/gameloop_test.go`
- `usecase/joingame_test.go`
- `usecase/kickinactiveuser_test.go`
- `repository/mock.go` (guard `SetPlayerStateCalled` add — small, see T03 scope)

## Notes / risks

- Do not add `time.Sleep` as the primary sync in any test; it's flaky under load.
- If a mock's `*Called` bool is needed, make it an `atomic.Bool` or guard with a mutex so
  both the production goroutine write and the test read are safe; still wait for completion
  via channel before asserting, to avoid reading mid-flight.