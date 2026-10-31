# prncompcalc — Claude Code Guide

prncompcalc is a Go command-line tool that calculates 3D-printing dimensional
compensation from `expected actual` measurement files. It uses only the standard
library.

## Project Layout

```text
cmd/prncompcalc/       — entry point: arguments, file handling, output
internal/compensation/ — parsing, the fit and the report
internal/version/      — the release version
```

Keep `cmd/` thin: logic belongs in `internal/` where it can be tested directly.

## Build & Test

Always use the Makefile. The Go toolchain is project-local in `.go/`, there is
no system `go`, and `make prepare` installs it.

```bash
make all       # build bin/prncompcalc
make test      # go test ./...
make lint      # golangci-lint
make fmt       # golangci-lint --fix
make ci-check  # everything CI runs, reporting both before failing
```

`GO_VERSION` and `GOLANGCI_LINT_VERSION` in the `Makefile` are the only place the
versions are pinned; CI installs through `make prepare`. Run `make distclean`
after changing one.

## Tests

Tests live beside the code in the same package, with no mocks. Tests write only
to `t.TempDir()`. `TestProgramExitCode` builds the real binary and finds `go` on
`PATH`, so run the tests through `make`.

## Algorithm

`README.md` describes the method. Two parts of it are deliberate:

- Do not go back to measured/expected ratios or medians of them. They count an
  offset as shrinkage, and that was the original bug.
- Do not derive hole compensation from outer dimensions, or add a pass/fail check
  on the largest residual. The first is unmeasured guesswork and the second
  fails good data more often as points are added.

The outlier constants in `compensation.go` were chosen by simulation.

## Linting

Config: `.golangci.yaml`. Run `make lint` before
finishing a task.

## Versioning

`Version` in `internal/version/version.go` is the single source of truth. Every PR bumps it, or
carries the `no-release` label if it ships nothing runnable. The tag is created
by a workflow after the merge, so never tag by hand. See `CONTRIBUTING.md`.

## What Not to Commit

`.gitignore` excludes `bin/` and `.go/`.
