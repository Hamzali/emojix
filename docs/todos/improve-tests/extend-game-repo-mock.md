# T03 — Extend MockGameRepository

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** T06, T07, T08, T09, T10 (every usecase test that touches the game repo)

## Problem

`repository/mock.go` `MockGameRepository` only stubs `GetPlayers`, `GetMessages`,
`GetScores`, `GetLatestTurn`, `AddPlayer`, `SetPlayerState`. It embeds the
`GameRepository` interface for everything else, which is **`nil`** at runtime. Any test that
exercises `Guess`, `Message`, `InitGame`, `Leaderboard`, `GameWord` currently can't be
written because the needed mocks panic.

`InitGame` also goes through `UnitOfWorkFactory.New().GameRepository()`, so the unit-of-work
side is untestable too.

## Goal

Make `MockGameRepository` (and the surrounding mocks) able to stand in for the entire
`GameRepository` interface, including the transactional path, so `Guess`, `Message`,
`InitGame`, etc. become testable in Phase 2.

## Approach

Add per-method mock fields following the existing pattern (`XMock func(...) (...)` +
optional `XCalled bool`). Embedding `GameRepository` stays so un-set methods remain
explicit-but-panicking (keeps the "you forgot to wire it" signal; tests set the ones they
need).

Methods to add a `Mock` field for:

- `FindByID(ctx, id) (model.Game, error)`
- `Create(ctx) (model.Game, error)`
- `AddTurn(ctx, gameID, wordID) (model.GameTurn, error)`
- `SendMessage(ctx, gameID, turnID, userID, content) (model.Message, error)`
- `AddScore(ctx, gameID, userID, messageID, turnID, score int) error`

Add `*Called` booleans only where a test would assert the call happened (mirroring
`AddPlayerCalled`); do not bloat — most assertions are via the mock funcs themselves.

### UnitOfWork mocks (new)

Add to `repository/mock.go`:

- `MockUnitOfWork` implementing `UnitOfWork` with `GameRepositoryMock *MockGameRepository`
  (or a getter), `RollbackMock`, `CommitMock`, and `RollbackCalled`/`CommitCalled`.
- `MockUnitOfWorkFactory` implementing `UnitOfWorkFactory` with
  `NewMock func(ctx) (UnitOfWork, error)`; default returns a `MockUnitOfWork`.

## Out of scope

- Concurrency guards on mock flags (partly T02). Keep it simple here; T02 will add the
  `atomic`/mutex once `Pub`-goroutine interactions are the focus.
- Mock coverage for `UserRepository` beyond `FindByID` (already there) and `CreateOrUpdate`
  (add only if a Phase-2 test needs it — `InitUser` does, so add a `CreateOrUpdateMock`).
- Mock coverage for `WordRepository` — `FindByID` exists; add `GetAllMock` (needed by
  `InitGame`/`newGameTurn`).

## Acceptance criteria

- [ ] `MockGameRepository` no longer panics on `Create/AddTurn/SendMessage/AddScore/FindByID`
      when those mocks are set.
- [ ] `MockUnitOfWork` and `MockUnitOfWorkFactory` exist and let `InitGame` run against
      fakes end-to-end (commit/rollback observable).
- [ ] `MockWordRepository` gains `GetAllMock`; `MockUserRepository` gains
      `CreateOrUpdateMock`.
- [ ] `go vet ./repository/...` and `go test ./repository/...` still pass.
- [ ] Existing tests remain unchanged/green.

## Files touched

- `repository/mock.go`

## Notes

- Embed the interface on the new mocks (as existing ones do) so unspecified methods panic
  loudly rather than silently nil-deref later. This is the established convention here.
- Don't add a generic "recording" mechanism; one `Mock` func per method, matching the
  existing style, keeps diffs reviewable.