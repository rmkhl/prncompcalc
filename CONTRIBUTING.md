# Contributing to prncompcalc

Run `make ci-check` before pushing. It runs everything CI runs.

## Versioning

prncompcalc is versioned with [semantic versioning](https://semver.org/). The
single source of truth is the `Version` constant in `internal/version/version.go`.

### Bump the version in the PR

| Bump      | When                                                              |
| --------- | ----------------------------------------------------------------- |
| **Patch** | Bug fixes only. Nothing about the interface or behaviour changes. |
| **Minor** | Functionality is added or changed, in a way users survive.        |
| **Major** | The input format, arguments or output format are invalidated.     |

The `Version bump` CI job fails a pull request whose `internal/version/version.go` is unchanged.
For a PR that ships nothing runnable, such as documentation or CI, add the
**`no-release`** label and the check is skipped. It re-runs when the label is
added or removed, so there is no need to push a commit to clear it.

### The tag is created for you

Do not create release tags by hand. When a change lands on master, the
`Release tag` workflow reads `internal/version/version.go` and pushes the matching
`vMAJOR.MINOR.PATCH` tag. A merge that did not bump the version finds the tag
already present and does nothing.

Tags are created after the merge on purpose: a tag pushed from a PR branch
would point at a commit that may never land.
