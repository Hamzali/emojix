# T08 — Test Guess (+ fix duplicate-guess bug)

- **Status:** `[ ]`
- **Depends on:** T03 (SendMessage/AddScore mocks + UoW), T04 (PubAll not strictly needed,
  but `GameMsgNotification`/`GameCorrectGuessNotification` use `Pub` which needs T04's
  race-safe mock)
- **Unblocks:** a real correctness fix lands behind a regression test

## Problem

`usecase/emojix.go` `Guess` has **no tests** and contains at least one latent bug:

```go
guessedCount := 1
for _, p := range players {
    for _, s := range scores {
        if s.PlayerID == p.ID && s.TurnID == turnID {
            guessedCount += 1
        }
    }
}
pointCoeff := len(players) / guessedCount   // integer division!
point := basePoint * pointCoeff
err = gameRepo.AddScore(...)
```

Issues a test will surface:

1. **Duplicate-guess bug.** When the same user guesses correctly twice, `guessedCount`
   jumps for every existing score row *plus* the implicit `+1`, a new `AddScore` row is
   inserted, and `guessedCount` can exceed `len(players)` making `pointCoeff` 0 (the user
   scores nothing, but the row is still written). Worse, `guessedCount == len(players)`
   then stops being true even when everyone has guessed, so `EndGameTurn` may never fire.
2. **Integer-division scoring.** `len(players)/guessedCount` floors — with 3 players and 2
   guesses this is `1`; with 3 and 1 it's `3`. The README's point system ("+5 base, +1/sec
   left, -1 wrong guess") does not match this formula at all. At minimum this drift is
   undocumented; a test will pin the *current* behavior so the later decision (align to
   README vs keep) is deliberate.
3. **`guessedCount == len(players)` triggered turn end** counts `players` from
   `gameRepo.GetPlayers` (all players, active+inactive?), while `GameState` filters active
   only. Mismatch — a left/inactive player would prevent turn end. Test should pin this.
4. **`Pub` calls fire unconditionally** even when commit/inserts fail (err isn't checked
   before `go e.gameNotifier.Pub` on the wrong-guess branch).

## Goal

Pin `Guess`'s current behavior with tests, then fix the duplicate-guess bug behind those
tests.

## Approach

### Tests first (red)

`TestGuess` sub-tests, all via mocks (T03/T04). Word fixed via `MockWordRepository.FindByID`
= `{Word: "Secret"}`. Latest turn fixed via `GetLatestTurn`. Use a channel for `Pub`
synchronization (T02 pattern).

1. **wrong guess** — content `!=` word → `SendMessage` called, `AddScore` **not** called,
   `GameMsgNotification` published with the raw content, `GameCorrectGuessNotification`
   **not** published, `EndGameTurn` **not** called, `Commit` called once.
2. **correct first guess** — single player guessed → `AddScore` called once with the
   expected point value (compute from the *current* formula and assert it; record the
   formula in a comment), both `GameMsgNotification` (with `"***"`) and
   `GameCorrectGuessNotification` published, `EndGameTurn` **not** called (guessedCount <
   len(players)).
3. **last correct guess ends turn** — set up so the current guess is the final one
   (scores exist for all other players on this turn) → `EndGameTurn("game-id")` called
   once.
4. **duplicate correct guess (bug)** — user already has a score on this turn and guesses
   the word again → **expected fixed behavior**: no second `AddScore`, no
   `GameCorrectGuessNotification`, no `EndGameTurn` (or a defined idempotent response).
   This test fails before the fix.
5. **`userRepo.FindByID` fails** — error propagated, no DB writes, no pub.
6. **`GetLatestTurn` fails** — same.
7. **`SendMessage` fails (commit error)** — error propagated, `AddScore` not called, no
   pub fires (verify the current code's ordering and fix if pub fires before commit —
   see issue 4 above; decide and pin via test).

### Fix

After the red tests, fix the duplicate-guess path. Recommended minimal fix:
- Before scoring, check the current user already has a row in `scores` for `turnID`. If so,
  treat as a no-op (return nil / a benign sentinel) — don't insert another score, don't
  pub a `guessed` notif. Keep the existing "pub a masked message" so other clients stay
  consistent, or skip entirely; pick one and test it.
- Drive `guessedCount` from `scores` only (drop the leading `+1`), and filter `players` to
  active only before comparing `guessedCount == len(activePlayers)`.

Leave the scoring *formula* as-is for this task (it's a separate decision — backlog),
only fix the duplicate/active-player bugs. Add a `// TODO:` referencing the README drift.

## Out of scope

- Aligning the scoring formula to README (backlog).
- Seaming `time.Now` (T13 — but Guess doesn't currently read time, so not blocked).

## Acceptance criteria

- [ ] All 7 sub-tests pass after the fix.
- [ ] The duplicate-guess test FAILS before the fix (commit it red the same day, then the
      fix in the same PR — per project TDD convention).
- [ ] `EndGameTurn` only fires on the truly-last active guess.
- [ ] No `Pub` fires after a failed commit.
- [ ] `go test -race ./usecase/...` green.

## Files touched

- `usecase/emojix_test.go` (new `TestGuess`)
- `usecase/emojix.go` (fix duplicate-guess + active-player count)

## Notes

- Keep the masked-content `Pub` on a correct guess consistent with the existing `***`
  message and the `guessed` notif — don't change the client contract in this task.
- If during the fix you find `len(players)` should use `filterActivePlayers`, reuse that
  method instead of duplicating the filter.