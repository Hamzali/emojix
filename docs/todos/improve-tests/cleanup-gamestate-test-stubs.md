# T06 — Clean up TestGameState stubs

- **Status:** `[ ]`
- **Depends on:** T03 (uses extended repo mocks)
- **Unblocks:** confidence for the masking/ordering logic in `GameState`

## Problem

`usecase/emojix_test.go` `TestGameState` has four `t.Run` blocks whose bodies are **empty**:

- `"should order messages from oldest to newest"`
- `"should sum up all scores in leaderboard"`
- `"turn should end when all players guessed the word"`
- `"should "` (clearly an abandoned idea)

Empty `t.Run` blocks **pass trivially and silently**, which is worse than no test: it looks
like coverage. They also encode a "todo" that nobody tracks.

## Goal

Either implement each as a real assertion or delete it. No empty `t.Run` remains in the file.

## Approach

Decide per stub:

1. **"should order messages from oldest to newest"** — implement. `GameState` reverses the
   repo message order ("newest first"). Set up 2–3 messages with deterministic order in
   `GetMessagesMock`, assert the resulting `gameState.Messages` order is reversed.
2. **"should sum up all scores in leaderboard"** — implement. Provide scores spanning
   multiple turns for the same player in `GetScoresMock`; assert one
   `LeaderboardEntry.Score` equals the sum. Adds coverage for `buildLeaderboard`'s
   `scoreMap` accumulation.
3. **"turn should end when all players guessed the word"** — implement, but **only the
   `allGuessed` branch**, not the timeout branch (timeout needs T13's clock). Provide
   scores for every active player matching `latestTurn.ID`; assert `TurnEnded == true`.
4. **"should "** — delete. It has no body and no identifiable intent.

If any stub turns out to be redundant with a T08/T10 test, delete it rather than duplicate
— but pick the deletion option explicitly, don't leave it empty.

## Out of scope

- The timeout branch of `TurnEnded` (`turnTimedOut := now.After(turnEndTime)`). That
  requires T13 (inject a Clock). Note it as a follow-up assertion to add after T13 lands;
  do not block T06 on T13.

## Acceptance criteria

- [ ] Zero empty `t.Run` bodies remain in `usecase/emojix_test.go`.
- [ ] The three implemented stubs assert the behavior in their names.
- [ ] `go test -race ./usecase/...` green.
- [ ] No assertion depends on `time.Sleep`.

## Files touched

- `usecase/emojix_test.go`

## Notes

- For "turn end when all guessed", `GameState` derives `allGuessed` from
  `Leaderboard[].GuessedWord` — so the test only needs `GetScoresMock` to return a score
  row for every active player on `latestTurn.ID`. No clock involved.
- After T13, add a 4th sub-test "turn should end when timer expires" using the fake clock;
  track that in T13's file as a follow-up.