# Improve Tests — Master Plan

Goal: close the testing gaps surfaced during the repo study. Make `go test -race -cover ./...`
green and meaningful, fix the real bugs the missing tests hide, and unblock fast iteration on
new features without manual verification.

Scope is bounded to testing + the correctness bugs the new tests expose. Pure feature work,
UI redesign, and content stay out. Each task is detailed in its own `*.md` file in this folder.

## Status legend

`[ ]` not started · `[~]` in progress · `[x]` done · `[!]` blocked

## Phases & tasks

### Phase 1 — Foundation: races and mocks (unblocks everything)

| ID  | Task file                         | Title                                                    | Depends on | Status |
|-----|-----------------------------------|----------------------------------------------------------|------------|--------|
| T01 | fix-gamenotifier-race.md          | Add mutex to GameNotifier (production data race)         | —          | `[x]`  |
| T02 | fix-test-races.md                 | Fix test-side data races (mock flags, call counters)     | T01        | `[x]`  |
| T03 | extend-game-repo-mock.md          | Extend MockGameRepository (SendMessage/AddTurn/…)       | —          | `[x]`  |
| T04 | extend-notifier-mock.md           | Add PubAllMock + concurrency safety to MockGameNotifier | T01        | `[x]`  |
| T05 | extend-gameloop-mock.md           | Capture OnTurnEndHandler in MockGameLoop (+ PubAll)      | —          | `[x]`  |

Phase 1 exit: `go test -race ./...` is green on existing tests and the usecase/repo/service
mocks can stand in for the full interface set.

### Phase 2 — Fill usecase gaps (depends on Phase 1)

| ID  | Task file                         | Title                                                    | Depends on | Status |
|-----|-----------------------------------|----------------------------------------------------------|------------|--------|
| T06 | cleanup-gamestate-test-stubs.md   | Implement or delete empty `t.Run` stubs in TestGameState | T03        | `[x]`  |
| T07 | test-init-game-and-user.md        | TestInitGame + TestInitUser (nickname, UoW, loop start)  | T03, T05   | `[x]`  |
| T08 | test-guess.md                     | TestGuess + expose/fix duplicate-guess scoring bug       | T03, T04   | `[x]`  |
| T09 | test-message.md                   | TestMessage                                              | T03, T04   | `[x]`  |
| T10 | test-leaderboard-and-game-word.md  | TestLeaderboard + TestGameWord                           | T03        | `[x]`  |
| T11 | test-gameupdates-paths.md         | Cover msg/guessed/turnended/left GameUpdates paths       | T04        | `[x]`  |
| T12 | test-onturnend.md                 | TestOnTurnEnd retry path                                | T05, T04   | `[x]`  |

Phase 2 exit: every `EmojixUsecase` method has at least happy-path + one failure-path test.

### Phase 3 — Make time-based logic testable

| ID  | Task file                         | Title                                                    | Depends on | Status |
|-----|-----------------------------------|----------------------------------------------------------|------------|--------|
| T13 | inject-clock-into-usecase.md      | Inject Clock into usecase for deterministic TurnEnded    | —          | `[x]`  |

Enables deterministic turn-timeout assertions for T06/T08.

### Phase 4 — Outer layers

| ID  | Task file                         | Title                                                    | Depends on | Status |
|-----|-----------------------------------|----------------------------------------------------------|------------|--------|
| T14 | server-httptests.md               | httptest server tests with mock usecase + mock view      | —          | `[x]`  |
| T15 | template-smoke-tests.md           | Render every template with sample params (no typos)      | —          | `[x]`  |

### Phase 5 — Infrastructure correctness

| ID  | Task file                         | Title                                                    | Depends on | Status |
|-----|-----------------------------------|----------------------------------------------------------|------------|--------|
| T16 | enable-foreign-keys-and-tests.md  | Enable `PRAGMA foreign_keys = ON` + FK-constraint tests  | —          | `[x]`  |
| T17 | migrator-tests.md                 | Test Migrator (apply/Up/Reset/Seed) + fix SeedCmd path  | —          | `[x]`  |

### Phase 6 — Process / gate (last)

| ID  | Task file                         | Title                                                    | Depends on  | Status |
|-----|-----------------------------------|----------------------------------------------------------|-------------|--------|
| T18 | add-race-and-cover-to-scripts.md  | Wire `go test -race -cover ./...` into a test script     | T01, T02    | `[x]`  |

## Dependency graph (compact)

```
T01 ─┬─> T02 ─────────────────────────────────┐
     ├─> T04 ─┬─> T08 ─┐                       │
     │        ├─> T11  │                       │
     │        └─> T12 ─┤                       │
     │                │                       │
T03 ─┬─> T06          ├─> (usecase coverage)   │
     ├─> T07 ──────────┤                       │
     ├─> T08 ──────────┤                       │
     ├─> T09 ──────────┤                       │
     └─> T10 ──────────┘                       │
                                              │
T05 ─┬─> T07                                  │
     └─> T12                                  │
T13  (standalone, enables T06/T08 assertions)  │
T14, T15, T16, T17 (standalone)               │
T18 <── T01, T02 ─────────────────────────────┘
```

## Backlog (documented, not scheduled)

Documented in `BACKLOG.md`:

- Remove dead `ParseData` no-op implementations or start using them (notification interface).
- Dedupe word-masking regex+logic shared between `GameState` and `GameWord`.
- Pin/align the `Guess` scoring formula to the README spec (point system section).
- Recompile `regexp.MustCompile(\w)` to a package-level var.
- Profile/escape-analyze comma-joined notification data format (a comma in nickname breaks
  `ParseData`); consider a small struct serialization or length-prefixed format.
- Add a `golangci-lint` config beyond `go vet`.

## Definition of done for the whole plan

- `go test -race -cover ./...` is green from a clean checkout.
- Every method of `EmojixUsecase`, `GameRepository`, `UserRepository`, `WordRepository`,
  `GameNotifier`, `GameLoop`, and `Migrator` has at least one happy-path and one
  failure-path test.
- HTTP handlers (`server.go`) are covered via `httptest` with mock dependencies.
- All `template/*.gohtml` render smoke-tested.
- `PRAGMA foreign_keys = ON` is enabled and FK integrity is asserted.
- Coverage baseline is recorded (the number goes into PLAN.md once T18 lands).
  - **Recorded baseline:** `74.2%` (`go tool cover -func=coverage.out | tail -1`) via `script/test.sh`.