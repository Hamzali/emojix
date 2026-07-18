# T07 — Test InitGame + InitUser

- **Status:** `[ ]`
- **Depends on:** T03 (UoW + extended game/word repo mocks), T05 (loop captures handler)
- **Unblocks:** — 

## Problem

`usecase/emojix.go` `InitGame` and `InitUser` are the entry points to the whole game, and
they have **no tests**:

- `InitGame`: creates a game, adds a player, creates the first turn (via `newGameTurn`,
  which calls `wordRepo.GetAll` then picks a random word), commits through a UnitOfWork,
  and starts the `GameLoop`. None of these interactions are verified.
- `InitUser`: generates a random 128-bit hex ID, generates a nickname from
  `animals × adjectives`, calls `userRepo.CreateOrUpdate`. The nickname generator and ID
  format are not asserted.

Because `MockGameRepository` was missing `Create`/`AddTurn`, and no UoW mock existed, these
tests genuinely couldn't be written before T03/T05.

## Goal

Lock in the contracts of `InitGame` and `InitUser` with happy-path + failure-path tests.

## Approach

### `TestInitGame`

Setup: `MockUnitOfWorkFactory` returns `MockUnitOfWork` whose `GameRepository()` returns a
`MockGameRepository` with `Create`, `AddPlayer`, `AddTurn` set; `MockWordRepository` with
`GetAll` returning a known word list; `MockGameLoop` with `StartMock` (assert called with
`gameID`, `turnDuration`) and `SetOnTurnEndHandler` captured (T05). Use a channel to
synchronize the `Start` assertion (T02 pattern) since `Start` is synchronous here (not a
goroutine), but keep the style consistent.

Cases:
1. **happy path** — returns a `Game` with a non-empty ID; `Create` mock asserts it received a
   fresh game ID; `AddPlayer` called with that game ID + the given user ID; `AddTurn` called
   with the game ID + a word ID drawn from the `GetAll` list; `Commit` called exactly once;
   `Rollback` **not** called; `gameLoop.Start` called **after commit** (the comment in
   `InitGame` calls this out explicitly — verify by setting `CommitMock` to only signal the
   loop-assert channel after `Commit` has run).
2. **`uow.New` fails** — `InitGame` returns the error, no repo/loop calls happen.
3. **`gameRepo.Create` fails** — error propagated, `AddPlayer`/`AddTurn` not called,
   `Commit` not called, `Rollback` called (deferred), `Start` not called.
4. **`AddTurn`/`AddPlayer` fails** — same rollback discipline; `Start` not called.
5. **empty word list** (`GetAll` returns `[]`) — `pickGameWord` would panic on
   `mathRand.Intn(0)`. Decide behavior: either return an error before picking (preferred,
   fix in this task) or document the precondition. Pick one and test it.

### `TestInitUser`

Cases:
1. **happy path** — ID is 32 hex chars (`hex.EncodeToString(16 bytes)`); nickname matches
   `^([A-Z][a-z]+){2}$` (capitalize of an adjective then an animal from the known slices).
   `CreateOrUpdate` called once with the generated ID + nickname.
2. **random-gen failure** — can't easily inject; skip unless we extract `rand.Read` behind
   a seam. Note as a follow-up, don't force it here.
3. **`CreateOrUpdate` fails** — error propagated, no retry.

## Out of scope

- Seaming `crypto/rand` / `math/rand` for deterministic ID/word selection. Fine to add a
  small `randSource` seam if it makes case 1 cleaner, but keep it minimal; otherwise just
  assert the *shape* of the outputs (length/format), not their values.
- Testing `onTurnEnd` (that's T12).

## Acceptance criteria

- [ ] `TestInitGame` covers all 5 cases above; the "Start after commit" ordering is
      explicitly asserted.
- [ ] `TestInitUser` asserts ID format, nickname format, and `CreateOrUpdate` call.
- [ ] If the empty-word-list case requires a code change, the change is in this task and
      tested.
- [ ] `go test -race ./usecase/...` green; `go vet` clean.

## Files touched

- `usecase/emojix_test.go` (new tests)
- `usecase/emojix.go` (only if empty-word-list behavior is fixed here)

## Notes

- `InitGame` calls `e.gameLoop.Start(context.Background(), game.ID, turnDuration)` with a
  **background** context deliberately (the request context may be cancelled when the
  response returns). Tests should assert the game ID passed, not the context identity.
- `turnDuration` is an unexported package const (`time.Second * 60`); tests reference
  `time.Minute` literally to avoid exporting it, or export it if a test wants exactness.
  Prefer literal `time.Minute` to avoid touching exports.