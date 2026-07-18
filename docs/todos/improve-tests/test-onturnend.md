# T12 — Test OnTurnEnd (retry + PubAll)

- **Status:** `[ ]`
- **Depends on:** T05 (MockGameLoop captures the handler so the test can fire it),
  T04 (PubAll is mockable)
- **Unblocks:** confidence in the turn-rotation failure recovery path

## Problem

`usecase/emojix.go` `onTurnEnd` is the single most complex untested code path:

```go
func (e *emojixUsecase) onTurnEnd(ctx, gameID) {
    e.gameNotifier.PubAll(gameID, &GameTurnEndNotification{})
    time.Sleep(5 * time.Second)
    err := e.newGameTurn(ctx, e.gameRepo, gameID)
    if err != nil {
        log.Printf(...)
        time.Sleep(time.Second)
        err = e.newGameTurn(ctx, e.gameRepo, gameID)
    }
    if err != nil {
        e.gameLoop.StopGame(gameID)
    }
}
```

- `PubAll` (needs T04) is exercised nowhere.
- The two-retry-then-`StopGame` recovery logic is exercised nowhere.
- `time.Sleep(5s)` and `time.Sleep(1s)` make this slow/flaky under test.

## Goal

Cover the dispatch + retry + stop behavior, deterministically and quickly.

## Approach

### Make sleeps controllable

The cleanest move is a seam: package-level `var turnEndWait = 5 * time.Second` and
`var turnRetryWait = time.Second` (or reuse whatever `Clock` seam T13 introduces — prefer
the `Clock` seam if T13 has landed, else add the two vars here and let T13 fold them in).
Tests override them to ~0. Avoid `time.Sleep` in test bodies entirely.

### Tests (after seam)

1. **happy path** — `PubAllMock` records one `&GameTurnEndNotification{}`;
   `newGameTurn` succeeds (`wordRepo.GetAll` returns ≥1 word, `gameRepo.AddTurn` returns
   nil). Assert: exactly one `PubAll`, exactly one `AddTurn`, **zero** `StopGame`.
2. **first add fails, retry succeeds** — `AddTurn` fails on first call, succeeds on second
   (use a call-counter in the mock). Assert: one `PubAll`, **two** `AddTurn`, zero
   `StopGame`. Wait interval was observed to be the (mocked) retry value — only assert
   ordering, not wall-clock.
3. **both retries fail** — `AddTurn` always errors. Assert: one `PubAll`, two `AddTurn`,
   **one** `StopGame("game-id")`.
4. **`wordRepo.GetAll` empty** — surfaces `pickGameWord`'s `Intn(0)` panic via `newGameTurn`
   to `onTurnEnd`. If T07 added the empty-list guard, this stays a clean error path: two
   `GetAll`/`AddTurn`-equivalent attempts then `StopGame`.

### Invoking the handler

The handler is installed by `NewEmojixUsecase(...)` via `gameLoop.SetOnTurnEndHandler`.
With T05 in place, the test:

```go
gl := &service.MockGameLoop{}
uc := usecase.NewEmojixUsecase(..., gl)
// gl.OnTurnEndHandler is now set
gl.OnTurnEndHandler(context.Background(), "game-id")
```

…then assert on the mocks. Synchronize via channels as usual.

## Out of scope

- Replacing the `5s` inter-turn sleep with a real game-design wait (production behavior
  decision; tests only make it fast).
- Testing that the real `GameLoop` calls `onTurnEnd` (that's `service/gameloop_test.go`).

## Acceptance criteria

- [ ] All 4 cases pass.
- [ ] Total test runtime added by `TestOnTurnEnd` < 100ms (waits are mocked near-zero).
- [ ] `StopGame` is called exactly when both retries fail, and only then.
- [ ] `go test -race ./usecase/...` green.

## Files touched

- `usecase/emojix_test.go` (new `TestOnTurnEnd`)
- `usecase/emojix.go` (add the two wait vars, or reuse T13's `Clock` seam — coordinate so
  only one seam lands)

## Notes

- Pick exactly one timing seam for the package (`Clock` from T13 preferred) to avoid two
  parallel override mechanisms. If T13 hasn't landed, add the two `var`s here and refactor
  to `Clock` when T13 lands. Add a cross-reference comment.