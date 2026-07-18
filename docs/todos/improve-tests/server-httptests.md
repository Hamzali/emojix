# T14 — Server httptest suite

- **Status:** `[ ]`
- **Depends on:** — (structurally independent; benefits from the mock seam conventions of
  T03/T04 but doesn't require them — a `MockEmojixUsecase` and `MockView` are introduced
  here)
- **Unblocks:** behavior of all HTTP routes can be verified without a browser

## Problem

`server.go` (421 LOC) is the largest file in the repo and the tested surface is one joke
test (`server_test.go` `TestAdd` returns `1+2==3`). Every handler does meaningful work:

- Cookie/session handling (`getSession`, `InitSession`, the `HttpOnly`/`Secure` split via
  `ENV`).
- Redirects (`302 Found` for `/game/join` to avoid the 301-cache bug the DEVLOG records).
- `Hx-Trigger` header on `/guess` (HTMX contract — a regression here silently breaks the
  client).
- `Sse` handler: `init` event, `text/event-stream` headers, `X-Accel-Buffering`, request
  `ResponseController`, delayed `KickInactiveUser` goroutine.
- Error rendering (`handleError`).

None of this is verified. The DEVLOG says server tests were deferred; the rest of the
plan now unblocks them.

## Goal

Cover every route handler with `httptest` using a `MockEmojixUsecase` + `MockView`. No live
HTTP server, no real DB, no real templates.

## Approach

### Mocks (new, in package `emojix` test file or `emojix/mocks_test.go`)

- `MockEmojixUsecase` implementing `usecase.EmojixUsecase` with per-method `func` fields +
  `*Called` counters. Follow the same embedding-the-interface convention if you want
  un-set methods to panic; given there are many methods and a server test sets all of
  them, a defensive no-op-default (like T04) is more practical here.
- `MockView` implementing `View` with per-render `func` fields that record their args and
  optionally write canned bytes to the `io.Writer`.

Because these are only used by server tests, keep them in `_test.go` files (package
`emojix_test` or `emojix`). Don't add them to a reusable package — the existing mock files
live in their own packages because they're consumed cross-package; these aren't.

### Handlers (route → expected behavior)

For each handler: a happy case (asserts status code, redirect target, header, the
render/view call args, the write body) and at least one failure case (usecase returns
error → `handleError` writes 500 + error template).

1. **`Index`** — has session → renders index with `Title`/`Nickname`; no session → 302 to
   `/init?from=`.
2. **`InitSession`** — calls `InitUser`, sets both cookies (`Path=/`, `HttpOnly`, plus
   `Secure` when `ENV=prod`), redirects to `from` (default `/`).
3. **`JoinGame`** — path id and query-id variants both mapped; on success 302 to
   `/game/{id}`; on `ErrJoinGameUserAlreadyJoined`/`ErrJoinGameRoomFull` → 500 (current
   behavior; pin it — note the UX gap as backlog rather than fix here).
4. **`NewGame`** — POST, 303 See Other to `/game/{id}`; session-missing → redirect.
5. **`Game`** — `TurnEnded` redirects 303 to `/game/{id}/loading`; otherwise renders
   `GamePageViewParam` with the expected fields incl. `MaskedWord` split.
6. **`LoadingGame`** — renders loading template; note the `TODO` about owner check is
   untested (backlog).
7. **`Message`** / **`Guess`** — parse form `content`; `Guess` sets `Hx-Trigger: guessed`;
   both render a `GameMsg` for the *current* user (assert `Me==true`, nickname from
   session).
8. **`Leaderboard`** / **`GameWord`** — render calls with the right params; error paths.
9. **`Sse`** — assert response headers (`Content-Type`, `Cache-Control`,
   `X-Accel-Buffering`), that an `init` event is flushed, then close the context (use
   `r.Context()` cancel) so `GameUpdates` returns and the handler exits; the
   `KickInactiveUser` goroutine is harder — inject a small clock seam or assert via
   dependency on `emojixUsecase.KickInactiveUser` being called after a wait. Pragmatic:
   accept the 30s wait seam from T13 (`Clock`) so the test advances the clock and asserts
   the kick call. If T13 isn't ready, stub `KickInactiveUser` to record the call and skip
   the real wait by injecting a near-zero duration via a new `kickInactiveDelay` package
   var (consistent with T12's seam choice).
10. **`handleError`** — via any failure case: 500 + the error template rendered.

## Out of scope

- Integration testing against a real DB (T16/T17 cover the DB layer).
- Real template rendering (T15 covers that; here `MockView` writes canned bytes).
- Testing the SSE client/JS behavior (no browser test in scope).

## Acceptance criteria

- [ ] Every route in `Start()` has at least one happy + one failure test.
- [ ] Cookie attributes (`HttpOnly`, `Secure`-on-prod) are asserted literally.
- [ ] `Guess` test asserts `Hx-Trigger: guessed`.
- [ ] `Sse` test asserts the three headers and the `init` event + clean exit on context
      cancel.
- [ ] `TestAdd` is deleted.
- [ ] `go test -race ./...` green.

## Files touched

- `server_test.go` (rewrite — remove `TestAdd`, add the suite)
- `emojix/mocks_test.go` (new — `MockEmojixUsecase`, `MockView`)

## Notes

- Handlers are registered on the default `net/http` mux via `http.HandleFunc("GET /…", …)`.
  For tests, call the method directly (`e.Index(w, r)`) rather than mounting the mux —
  simpler and the route patterns are already implicitly covered. If you want to also
  verify routing, one test can spin `httptest.NewServer` over the registered mux; keep it
  to a smoke test to avoid coupling.
- `r.PathValue("id")` works only when the request is matched by a pattern-registered mux.
  For direct-handler tests, construct the request with `r.SetPathValue("id", "...")`
  (Go 1.22+) so `PathValue` returns the value.