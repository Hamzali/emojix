# T10 — Test Leaderboard + GameWord

- **Status:** `[ ]`
- **Depends on:** T03 (`GetPlayers`/`GetScores`/`GetLatestTurn` already mockable; `FindByID`
  word mock already exists)
- **Unblocks:** —

## Problem

`Leaderboard` and `GameWord` usecases (the partial-HTML render paths behind
`GET /game/{id}/leaderboard` and `GET /game/{id}/word`) have **no tests**, even though
their logic (masking, guessed detection, active-player filtering) is non-trivial and
partly duplicates `GameState` (the TODOs already flag this duplication as techdebt).

Bugs to check for:
- `GameWord` recomputes `regexp.MustCompile(\w)` on every call (perf, not correctness).
- `Leaderboard` and `GameState` reimplement the same `buildLeaderboard`; divergence is
  invisible without a test on `Leaderboard` itself.
- Both reject a caller who isn't an active player (`isPlayerInGame`) — no test pins the
  error.

## Goal

Pin `Leaderboard` and `GameWord` behavior with focused unit tests.

## Approach

### `TestLeaderboard` (usecase method)

1. **happy path** — multiple active players, scores across turns → entries sorted/ordered
   same as `GameState` leaderboard; `Me` set on the matching `currentUserID`; `GuessedWord`
   reflects scores on the latest turn only.
2. **user not in game** — `isPlayerInGame` returns error → propagated, empty slice
   (current behavior) returned.
3. **inactive players excluded** — provide one `Inactive` player → not in the result.
4. **`GetPlayers` fails** — error propagated.
5. **`GetScores` fails** — error propagated.
6. **`GetLatestTurn` fails** — error propagated.

### `TestGameWord` (usecase method)

1. **not guessed** — no score for `currentUserID` on latest turn → returns masked word
   (each word char → `*`).
2. **guessed** — score exists for `currentUserID` on latest turn → returns raw word.
3. **word with non-`\\w` chars** (e.g. spaces, emoji) — verify what leaks unmasked (this
   documents the current regex limitation; add `// TODO:` backlog note about a better
   masking scheme). No fix; just pin.
4. **`GetLatestTurn` fails** — error propagated, empty string.
5. **`wordRepo.FindByID` fails** — error propagated.
6. **`GetScores` fails** — error propagated.

## Out of scope

- Deduping the `buildLeaderboard`/masking logic with `GameState` (backlog, but a test on
  `Leaderboard` is the prerequisite that makes the refactor safe).
- Recompiling the regex (backlog perf).

## Acceptance criteria

- [ ] All sub-tests pass.
- [ ] At least one sub-test per usecase exercises an error-propagation path from each
      repo call it makes.
- [ ] `go test -race ./usecase/...` green.

## Files touched

- `usecase/emojix_test.go`

## Notes

- The `Me`/`GuessedWord` assertions on `Leaderboard` mirror the ones `assertGameState`
  already does; reuse `assertValueError` rather than rewriting the helper.