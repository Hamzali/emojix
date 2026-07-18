# T18 — Wire `-race` and `-cover` into a test script

- **Status:** `[ ]`
- **Depends on:** T01, T02 (only land *after* those, so the script is green by default —
  otherwise it'll just red-screen until they're done)
- **Unblocks:** the discipline that makes every task above stick

## Problem

- Neither `script/run-dev.sh`, `script/run.sh`, nor any of the docs run `go test` with
  `-race`. The `service` package has a **production** data race that has shipped because no
  one runs `-race` routinely (T01).
- There's no coverage measurement anywhere, so the empty `t.Run` stubs in T06 looked like
  coverage for who-knows-how-long.
- `AGENTS.md` mandates "validate your changes using … vet, fmt and test" but says nothing
  about `-race` or coverage. The mandate is read literally as plain `go test`.

## Goal

A single script (`script/test.sh`) that runs `go vet`, `gofmt -l` (fail if any output),
`go test -race -cover ./...`, and prints the coverage profile summary. Plus a one-line
update to `AGENTS.md` pointing at it and naming `-race`/coverage.

## Approach

1. `script/test.sh`:
   ```sh
   #!/usr/bin/env sh
   set -e
   echo "fmt check"
   [ -z "$(gofmt -l .)" ] || { echo "gofmt needed:"; gofmt -l .; exit 1; }
   echo "vet"
   go vet ./...
   echo "test (race + cover)"
   go test -race -cover -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out | tail -1
   ```
   Match the existing script style (`#!/usr/bin/env sh` works for `run.sh`; keep `run-dev.sh`
   zsh-specific since it uses zsh globbing).
2. Add `coverage.out` to `.gitignore` so the profile isn't committed.
3. Update `AGENTS.md` "implementation" section with one sentence:
   *"Run `script/test.sh` before committing; it runs `gofmt`, `go vet`, and `go test -race
   -cover ./...`."* — keep AGENTS.md lean per its own instruction.
4. Update `doc/TODOs.md` techdebt section to remove "add e2e tests" duplication once this is
   the canonical test entrypoint (e2e is *not* in scope here; just stop double-tracking
   the baseline).

## Out of scope

- CI integration (no CI configured in the repo; adding one is its own task and user
  preference). The script is the building block for any future CI.
- Coverage thresholds. Just record the baseline number in PLAN.md once green; thresholds
  can come later.

## Acceptance criteria

- [ ] `script/test.sh` exists, is executable, and passes on a clean tree (assumes T01/T02
      landed).
- [ ] `gofmt -l .` empty.
- [ ] `go test -race -cover ./...` green; per-package coverage printed.
- [ ] `coverage.out` is gitignored.
- [ ] `AGENTS.md` references the script and mentions `-race`+coverage in one sentence.
- [ ] The aggregate coverage number is appended to the "Definition of done" section of
      `PLAN.md` as the recorded baseline.

## Files touched

- `script/test.sh` (new)
- `.gitignore` (add `coverage.out`)
- `AGENTS.md` (one sentence)
- `PLAN.md` (baseline number)

## Notes

- If `go test -race ./...` still red from a not-yet-landed Phase-2/4/5 task, don't block
  this task — but the script should fail loudly, which is the point. Land this task last
  per the dependency so the tree is green when it lands.
- `gofmt -l` returning filenames (non-empty) is the failure signal; the script prints them
  before exiting.