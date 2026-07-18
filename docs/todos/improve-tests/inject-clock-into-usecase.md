# T13 — Inject Clock into the usecase

- **Status:** `[ ]`
- **Depends on:** — (standalone refactor)
- **Unblocks (deterministic assertions for):** T06 ("timer expires" sub-case), T08 if Guess
  ever reads time, and any future turn-timer UI test.

## Problem

`usecase/emojix.go` `GameState` computes the turn-timeout branch with real wall clock:

```go
turnEndTime := gameState.TurnStartedAt.Add(turnDuration)
now := time.Now()
turnTimedOut := now.After(turnEndTime)
gameState.TurnEnded = allGuessed || turnTimedOut
```

Meanwhile the game *loop* already has a `service.Clock` abstraction (`Clock.After(d)` and
`FakeClock`/`RealClock`) precisely to make timing deterministic — but the usecase doesn't
use it, so `TurnEnded`-by-timeout is impossible to test without `time.Sleep(turnDuration)`.

`onTurnEnd` also hardcodes `time.Sleep(5 * time.Second)` and `time.Sleep(time.Second)`
(T12 calls this out). Both should flow through the same seam.

## Goal

One `Clock` abstraction consumed by both `service` and `usecase`, so all time-based code is
deterministic.

## Approach

1. Promote the `Clock` seam. Currently `service.Clock` only has `After(d)`. Add `Now()
   time.Time` to the interface (implement on `RealClock` → `time.Now()`, and `FakeClock`
   already has `Now()`, just promote the method to satisfy the interface).
2. Add a `clock Clock` field to `emojixUsecase`; inject it through `NewEmojixUsecase`
   (new optional param, or extend the existing constructor). Wire `RealClock` in
   `cmd/server/main.go`.
3. Replace `time.Now()` in `GameState` with `e.clock.Now()`.
4. Replace `time.Sleep(...)` in `onTurnEnd` with `<-e.clock.After(...)` (deterministic:
   `FakeClock.Advance` fires it).
5. Make `turnDuration` and the inter-turn wait/retry waits injectable too, OR keep
   `turnDuration` as a const and only seam the sleeps — prefer seams for the two sleeps
   (T12's `var`s become `Clock.After`-based). Coordinate with T12 so only one mechanism
   lands.
6. Expose a constructor or test helper that builds a usecase with a `FakeClock`.
   `NewEmojixUsecaseForTest`-style is acceptable but prefer just exposing the constructor
   with the clock arg and letting tests pass a `FakeClock`.

## Out of scope

- Changing `turnDuration` (60s) — production tuning.
- Seaming `time.Now` reads in the repository layer (DB uses its own `time.Now` for
  created_at; testing those is separate, T16/T17 territory).

## Acceptance criteria

- [ ] `service.Clock` has `Now() time.Time`; `FakeClock.Now()` satisfies the interface.
- [ ] `emojixUsecase` reads time and sleeps only through `Clock`.
- [ ] A new `TestGameState_TurnTimedOut` sub-test (or T06 follow-up) advances a `FakeClock`
      past `turnDuration` and asserts `TurnEnded == true`, with no `time.Sleep`.
- [ ] `cmd/server/main.go` wires `RealClock` and behavior is unchanged.
- [ ] `go test -race ./...` green; `go vet` clean.

## Files touched

- `service/gameloop.go` (extend `Clock`)
- `service/fakeclock.go` (make `Now()` satisfy `Clock`)
- `usecase/emojix.go` (add field, constructor param, replace `time.Now`/`time.Sleep`)
- `cmd/server/main.go` (pass `RealClock`)
- `usecase/emojix_test.go` (new deterministic timeout test; update existing constructor
  calls to pass a clock — most can use `RealClock` since they don't touch time, or
  `NewFakeClock` if simpler)

## Notes

- Touches every `NewEmojixUsecase(...)` call site (existing tests + main). Most existing
  usecase tests pass `&service.MockGameLoop{}`; they can pass `service.NewRealClock()` to
  keep their assertions unchanged, since they don't exercise time. Make this the easy
  default.
- Keep `turnDuration` unexported; tests reference `time.Minute` or import the const if you
  choose to export it. Prefer not exporting.