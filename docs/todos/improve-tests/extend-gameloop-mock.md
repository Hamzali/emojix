# T05 — Extend MockGameLoop

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** T07, T12

## Problem

`service/gameloop_mock.go` `MockGameLoop.SetOnTurnEndHandler` accepts the handler and
stores it in the local `SetOnTurnEndHandlerCalled` bool but **discards the handler
itself**. `NewEmojixUsecase` uses `gameLoop.SetOnTurnEndHandler(func(ctx, gameID){ … })`
to wire the usecase's `onTurnEnd` callback; in tests that handler is silently lost, so
nothing can trigger turn-end behavior through the loop. `T12` (TestOnTurnEnd) is
consequently impossible.

## Goal

Capture the handler installed by `NewEmojixUsecase` so tests can (a) assert it was
installed, and (b) invoke it manually to drive `onTurnEnd` deterministically.

## Approach

1. Add `OnTurnEndHandler OnTurnEndHandler` field to `MockGameLoop`.
2. In `SetOnTurnEndHandler`, set both `SetOnTurnEndHandlerCalled = true` and
   `m.OnTurnEndHandler = handler`.
3. Convenience method `FireOnTurnEnd(ctx, gameID)` that calls the captured handler if non-nil
   (used by T12).
4. Keep the existing per-method `*Mock`/`*Called` fields untouched; only add to them.

## Out of scope

- Defaulting unset mocked methods to no-op (the interface-embedding panic is intentional
  here; only `MockGameNotifier` (T04) gets defensive defaults because its goroutine
  callers make a panic harder to attribute).

## Acceptance criteria

- [ ] `MockGameLoop` retains the handler passed to `SetOnTurnEndHandler`.
- [ ] A test can call `mock.FireOnTurnEnd(ctx, "g1")` to trigger the usecase's `onTurnEnd`.
- [ ] `NewEmojixUsecase(...)` no longer "swallows" the handler in tests — verified by a
      short unit test in `service/gameloop_mock_test.go` or via T12.
- [ ] `go vet ./service/...` clean; existing loop tests unaffected.

## Files touched

- `service/gameloop_mock.go`

## Notes

- `OnTurnEndHandler` is already exported from `service/gameloop.go`, so the field type is
  available without new exports.
- Do not call the handler from inside `SetOnTurnEndHandler` — only store it, to preserve
  the real loop's "called once before Start" contract.