# T09 — Test Message

- **Status:** `[ ]`
- **Depends on:** T03 (GetLatestTurn/SendMessage mocks), T04 (Pub mock race-safe)
- **Unblocks:** —

## Problem

`usecase/emojix.go` `Message` has **no tests**. It:

1. Reads the latest turn (`GetLatestTurn`).
2. Reads the user (`userRepo.FindByID`).
3. Calls `gameRepo.SendMessage(...)` with `gameID, turn.ID, userID, content`.
4. Fires `go e.gameNotifier.Pub(gameID, userID, &GameMsgNotification{...})` with raw
   content — note it does NOT mask the content even if the content equals the secret word
   (potential leak: a user can type the actual word in the chat and it is broadcast
   unmasked — `Guess` is the only path that masks). That may or may not be a bug; a test
   pins the behavior.

Note also: `Message` does **not** use a UnitOfWork, while `Guess` does. Inconsistent but
out of scope here.

## Goal

Happy-path + failure-path coverage for `Message`, pinning the current (unmasked) behavior.

## Approach

`TestMessage` sub-tests (mocks from T03/T04, channel-synchronized `Pub`):

1. **happy path** — `SendMessage` called with the right `gameID`/`turnID`/`userID`/
   content; `GameMsgNotification` pub'd with the raw content and the user's nickname;
   `ParseData` round-trips back to the same `(UserID, Nickname, Content)`.
2. **content equals the secret word** — set `GetLatestTurn` → `wordRepo.FindByID` returns
   `{Word: "Secret"}`; call `Message(..., "Secret")` → assert `Pub` publishes
   `"Secret"` **unmasked** (pin current behavior; add a `// TODO:` referencing the masking
   gap so it's tracked without leaving it implicit).
3. **`GetLatestTurn` fails** — error propagated, no `SendMessage`, no `Pub`.
4. **`userRepo.FindByID` fails** — error propagated, no `SendMessage`, no `Pub`.
5. **`SendMessage` fails** — error propagated, no `Pub`.

## Out of scope

- Masking chat content that matches the word. This is a real game-integrity bug but is a
  behavior decision, not purely testing. Add a backlog note instead and a `// TODO:` at
  the test, don't fix here.
- Unifying the SendMessage/UoW inconsistency vs `Guess`.

## Acceptance criteria

- [ ] All 5 sub-tests pass.
- [ ] `Pub` assertions use a channel (no flag-after-sleep).
- [ ] `go test -race ./usecase/...` green.

## Files touched

- `usecase/emojix_test.go`

## Notes

- `Message` reads `userRepo.FindByID` purely to populate `Nickname` in the notification —
  so case 4 also documents that nickname is required for the pub payload.