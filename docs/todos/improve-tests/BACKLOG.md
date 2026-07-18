# Backlog — improve-tests scope-adjacent ideas

Documented but **not** scheduled tasks under this todo set. Each was raised during the repo
study, is worth tracking, but was deliberately kept out of the active PLAN.md because it's
either behavior design (not purely testing) or low-priority polish once the plan lands.

## B1 — Clean up the GameNotification.ParseData dead code

Five notification types implement `ParseData(data string) error`. Four of them
(`GameJoinNotification`, `GameCorrectGuessNotification`, `GameTurnEndNotification`,
`UserLeftNotification`) return `nil` and ignore `data`. Only `GameMsgNotification` parses.
The interface forces the no-op. Either:

- Remove `ParseData` from the `GameNotification` interface and call parsing directly where
  `GameMsgNotification` needs it (in `server.go` `Sse`), or
- Use it consistently (parse on the consumer side for every type) so the types carry their
  own deserialization.

The decision is a design call — `server.go` currently calls `msgNotif.ParseData(data)` for
`"msg"` and treats everything else as opaque `data` strings. Pick once and apply.

## B2 — Dedupe word-masking between GameState and GameWord

`GameState` and `GameWord` both:
- build a score map keyed by `PlayerID`+`TurnID`,
- detect whether the *current* user has guessed,
- run `regexp.MustCompile(\w)` and `ReplaceAllString(word, "*")`.

`doc/TODOs.md` already flags "fetch all game data with one sql query" and the
"Leaderboard/GameState duplication" techdebt (Jan 8 entry). A clean extraction (a
`WordMasker` type or reuse of `GameState`'s already-built `LeaderboardEntry.GuessedWord`)
would let `GameWord` derive from `GameState` rather than re-querying. Land this *after* T10
pins `GameWord`'s behavior, so the refactor is safe.

## B3 — Align the Guess scoring formula to the README

`README.md` documents a point system (guesser `+5` base, `+1/sec left`, `-1/wrong`; teller
`+1/guess`, `+1/sec`, `+5/all-guessed`). `usecase.emojix.go` `Guess` uses
`basePoint=10 * (len(players)/guessedCount)` integer division. The drift is undocumented.
T08 pins the *current* formula; the *alignment* to README is a product decision (do you
take time-left into account? wrong-guess penalty?). Decide → rewrite `Guess` → update T08
assertions. Track separately.

## B4 — Hoist the masking regex to package scope

`regexp.MustCompile(\w)` is recompiled on every `GameState` and `GameWord` call. Trivial
win: a `var wordMaskRegex = regexp.MustCompile(\w)` at package scope. Tiny task; fold into
T10 or B2 rather than a standalone.

## B5 — Fix notification data encoding (commas break ParseData)

`GameMsgNotification.GetData` uses `fmt.Sprintf("%s,%s,%s", …)` and `ParseData` does
`strings.Split(data, ",")`. A nickname or message containing a comma mis-parses. The SSE
handler strips `\n` only. Options: a small struct serialization (json — but the repo avoids
deps; `encoding/json` is stdlib so acceptable), length-prefixed fields, or a rare sentinel
delimiter. Needs a decision (T11 documents the current contract; this changes it).

## B6 — Add `golangci-lint` config beyond `go vet`

`go vet` catches little. A minimal `.golangci.yml` enabling `staticcheck`, `ineffassign`,
`unused`, `errcheck` would catch the swallowed `defer tx.Rollback()` errors noted in T17
and unused exports like `assertCalledWithMsg`. Optional; the no-deps ethos is satisfied
since golangci-lint is a dev tool, not a runtime dependency.

## B7 — Real e2e browser/integration tests

`doc/TODOs.md` defers e2e "until you nail down the initial version". T14 (server
httptest) gives most of the value without a browser. A Playwright/Chromium pass is only
worth it once the UI stabilizes per the Jan 19 design direction. Not scheduled here.

## B8 — Channel cleanup strategy

`doc/TODOs.md`: "cleanup/close realtime channels for users properly or at least figure out
if they are cleaned up by go runtime". T01 makes the notifier thread-safe; the leak
question (whether long-lived games hold subscriber goroutines) is a runtime-scaleability
task, separate from this plan.

## B9 — Seaming `crypto/rand` / `math/rand` for deterministic IDs

`InitUser` (`generateRandomID` + `generateNickname`) and `pickGameWord` use package-level
`rand`. T07 tests the *shape* of outputs, not exact values. If exact determinism becomes
valuable (reproducible failing tests), seam both behind an injectable source. Low priority.

## B10 — Unify UoW usage between Guess (transactional) and Message/onTurnEnd (not)

`Guess` wraps writes in a UnitOfWork; `Message` and `onTurnEnd`→`newGameTurn` don't. Is the
difference intentional? Make the choice explicit and uniform — but it's a behavior
decision, so it lives here, not in the test plan.