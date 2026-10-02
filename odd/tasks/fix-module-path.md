# Fix the Go module path

## Problem

`go.mod:1` declared `module github.com/ezzek/skill-sync`, but the repository is
`github.com/eezzekl/skill-sync` — the final `l` was missing. `go install` had
therefore never worked for any published version:

```
$ go install github.com/eezzekl/skill-sync/cmd/skill-sync@v1.0.0
go: github.com/eezzekl/skill-sync/cmd/skill-sync@v1.0.0: version constraints conflict:
	module declares its path as: github.com/ezzek/skill-sync
	        but was required as: github.com/eezzekl/skill-sync
```

Present since the initial commit `dac6be6` (`git log -L1,1:go.mod`), so v1.0.0,
v0.1.0 and v0.2.0 all ship the wrong path. Homebrew, Scoop and direct tarball
downloads are unaffected: they distribute prebuilt binaries.

`SECURITY.md:16` carries the same typo in the private vulnerability reporting
URL, which makes that link a 404.

## Why no existing test caught it

The wrong path is internally *consistent*: every import agrees with `go.mod`, so
the compiler and the entire suite are happy. Only an external consumer resolving
the module through the proxy observes the mismatch. That is the invariant worth
pinning with a guard test.

## Testability

This is a build-level identity change, not a behavior change, so most of it is
verified by compilation plus the existing suite. One part *is* meaningfully
test-first: a guard asserting that the declared module path equals the canonical
repository path. That test fails today and would have caught the bug on day one.

## Tasks

- [x] 1. RED: guard test asserting `go.mod` declares the canonical module path.
- [x] 2. GREEN: rewrite the module path in `go.mod` and all 93 occurrences.
- [x] 3. Fix the broken `SECURITY.md` advisory URL.
- [x] 4. Update the import examples in the openspec design/tasks documents.
- [x] 5. Verify: full suite with `-race`, `go vet`, `gofmt`, cross-compile matrix.
- [ ] 6. Release `v1.1.0` so the proxy's `@latest` moves off `v1.0.0`.

## Note on `v1.0.0`

Deleting the `v1.0.0` tag cannot fix the version ordering: proxy.golang.org has
permanently cached `v1.0.0`, `v0.1.0` and `v0.2.0`, and its `@latest` resolves to
`v1.0.0`. Only publishing a semver-greater tag moves it. Hence `v1.1.0`.

## Evidence

**RED (task 1)** — `go test ./cmd/skill-sync/`:

```
--- FAIL: TestGoModDeclaresCanonicalModulePath
    go.mod module path mismatch:
      declared:  github.com/ezzek/skill-sync
      canonical: github.com/eezzekl/skill-sync
```

**GREEN** — 35 files rewritten, 0 stale references left in tracked files.
`go.mod` diff is the single module line; `go.sum` untouched.

- `go test -race ./...` — 19 packages pass, 0 failures
- `go vet ./...` — clean
- `gofmt -l` — clean
- Cross-compile — all 6 GoReleaser targets build
- `--version` wiring intact (`skill-sync version 1.1.0-rc` from ldflags)

**Process note:** the bulk `sed` also rewrote this document, which quotes the old
broken path as evidence, leaving self-contradictory text ("declares
`.../eezzekl/...` but the repository is `.../eezzekl/...` — the final `l` is
missing"). The evidence block was restored by hand. A blind repository-wide
replacement will corrupt any document that intentionally quotes the old value.
