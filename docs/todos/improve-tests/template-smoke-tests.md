# T15 — Template render smoke tests

- **Status:** `[ ]`
- **Depends on:** —
- **Unblocks:** catches template regressions before a request does

## Problem

`view.go` parses every `template/*.gohtml` from relative paths at `NewHTMLView()` time, and
`template.Must` will **panic** at startup if any file is missing or malformed. There are 11
gohtml files and no test parses or executes them. A typo in a template (e.g. `{{ .Foo}}`
referencing a non-existent field) only surfaces the first time someone hits the route —
during manual playtesting, which is exactly the slow feedback loop the DEVLOG complains
about.

## Goal

Render every template with sample params and assert no error. Catch shape/typo regressions
in one `go test`.

## Approach

1. A `_test.go` in package `emojix` calls `NewHTMLView()` and asserts it doesn't panic /
   error (this alone catches parse-time syntax errors).
2. A table of `render` calls, one per `View` method, each with representative params:
   - `renderErrorPage` — nil params.
   - `renderIndexPage` — `IndexPageViewParam{Title:"x", Nickname:"y"}`.
   - `renderGamePage` — `GamePageViewParam` with non-empty leaderboard (≥2 entries), ≥2
     messages, a multi-char `MaskedWord`, an `EmojiHint`, and a `TurnStartedAt`. Include
     the `"Me"` and `"GuessedWord"` branches.
   - `renderGameWord`, `renderGameMsg`, `renderGameLeaderboard`, `renderGameLoadingPage`
     — minimum-viable params.
3. For each, write to a `bytes.Buffer` and assert the buffer is non-empty and contains at
   least one expected substring (e.g. the nickname, or a known element id / template text).
   Keep asserts loose — the goal is "render succeeds and isn't blank" not "exact HTML" (exact
   HTML would couple the test to design changes).

### Running-path caveat

`NewHTMLView` hardcodes `"template/base.gohtml"` etc. — relative to the *current working
directory*. Tests must run from the repo root. Document this in a comment and consider
adding a `t.Helper`/`os.Chdir` guard that fails fast with a clear message if
`template/index.gohtml` isn't found.

## Out of scope

- Visual / CSS / layout correctness (no browser).
- Asserting specific HTML structure that would couple to redesigns.
- Fuzzing template inputs (out of scope; backlog).

## Acceptance criteria

- [ ] `NewHTMLView()` parses without error in a test.
- [ ] Every `View` render method is called at least once with sample params and returns
      no error.
- [ ] Each rendered buffer is non-empty and contains one representative expected
      substring.
- [ ] Test fails with a clear message if run from the wrong directory.
- [ ] `go test -race ./...` green.

## Files touched

- `view_test.go` (new)

## Notes

- If `NewHTMLView` is the only thing that references the template paths, a future
  refinement (backlog) could embed templates via `//go:embed template/*.gohtml` to remove
  the cwd dependency — but that changes the dev loop (`entr`-based restarts) and is out of
  scope here. The smoke test just makes the relative-path fragility loud and early.