# T11 — Test GameUpdates notification paths

- **Status:** `[ ]`
- **Depends on:** T04 (Pub/PubAll mock correctness, not strictly required since this test
  injects directly via the `Sub` mock channel, but T04 keeps the surrounding mock sane)
- **Unblocks:** —

## Problem

`usecase/emojix_test.go` `TestGameUpdates` only exercises the `"join"` notification. The
other four notification types emitted by the usecase are untested end-to-end through
`GameUpdates`:

- `"msg"` (`GameMsgNotification`)
- `"guessed"` (`GameCorrectGuessNotification`)
- `"turnended"` (`GameTurnEndNotification`)
- `"left"` (`UserLeftNotification`)

The handler signature (`notifType, data` strings) and the downstream SSE formatting live
in `server.go`; this test only verifies the usecase's **dispatch** is faithful
(type/data pass-through) and that context cancellation returns cleanly.

It will also surface which `ParseData` implementations are no-ops (see BACKLOG) by making
the round-trip behavior explicit.

## Goal

Cover the dispatch for every notification type + the cancellation path.

## Approach

Extend `TestGameUpdates` (already present, `join` case) into a table of cases. The shared
harness: `MockGameNotifier.SubMock` returns a test-owned `chan GameNotification` plus a
cleanup that flips a bool; the test pushes a notif on the channel, the handler records
`(type, data)`, then cancels the context.

Cases:
1. `"join"` — already exists; keep/refactor into the table.
2. `"msg"` — push `&GameMsgNotification{UserID, Nickname, Content}`; assert
   `notifier.GetType()==` `"msg"` and `GetData()==` `fmt.Sprintf("%s,%s,%s", ...)`.
3. `"guessed"` — `&GameCorrectGuessNotification{UserID, Nickname}` → `getData == "id,nick"`.
4. `"turnended"` — `&GameTurnEndNotification{}` → `data == ""`.
5. `"left"` — `&UserLeftNotification{UserID}` → `data == userID`.
6. **handler returns error** — push a notif, make the handler return an error → `GameUpdates`
   returns that error, cleanup ran.
7. **context cancellation** — cancel `ctx` without pushing anything → `GameUpdates`
   returns `nil`, cleanup ran once.

Pre-seed the channel from a goroutine or synchronously before calling; prefer synchronous
push where possible to avoid races, since the test channel is unbuffered and `GameUpdates`
only reads inside the loop.

## Out of scope

- Testing the SSE HTML formatting (that's `server.go` — T14).
- Round-tripping `ParseData` as part of `GameUpdates` (it isn't called here; it's used only
  in the SSE handler). Pin `ParseData` behavior separately if desired — backlog.

## Acceptance criteria

- [ ] All 7 cases pass.
- [ ] Cleanup is asserted to run exactly once per `GameUpdates` return (both normal and
      error and cancellation).
- [ ] `go test -race ./usecase/...` green; no `time.Sleep` for the cancellation case (use a
      channel or `cancel()` directly).

## Files touched

- `usecase/emojix_test.go`

## Notes

- The empty/`""` data for `turnended` and the comma-joined data for `msg`/`guessed` are
  contracts the SSE handler in `server.go` depends on — keep this test as the canonical
  record of those formats so any refactor here is forced to update the server test (T14).