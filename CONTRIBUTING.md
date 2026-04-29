# Contributing to skill-sync

Thanks for contributing. skill-sync enforces a strict **issue-first workflow** — every change starts with an approved issue.

---

## Contribution Workflow

```
Open Issue → Get status:approved → Open PR → Add type:* label → Review & Merge
```

### Step 1: Open an Issue

Use the correct template:
- **Bug Report** — for bugs
- **Feature Request** — for new features or improvements

> Blank issues are disabled. You must use a template.

Fill in all required fields. Your issue will automatically receive the `status:needs-review` label.

### Step 2: Wait for Approval

A maintainer will review the issue and add `status:approved` if it's accepted for implementation.

**Do not open a PR until the issue is approved.**

### Step 3: Open a Pull Request

Once the issue is approved:

1. Fork the repo and create a branch from `main`
2. Implement your change
3. Open a PR using the PR template — **link the approved issue** with `Closes #N`
4. Add exactly **one `type:*` label** to the PR

### Step 4: CI Checks

Two checks run automatically on every PR:

| Check | What it runs |
|-------|-------------|
| **Unit Tests** | `go test ./...` |
| **Coverage** | `go test -cover ./...` |

All checks must pass before a PR can be merged.

---

## Running Tests Locally

```sh
# Run all tests
go test ./...

# With coverage
go test -cover ./...
```

No build tags, no extra setup. Tests use real temporary directories — avoid mocking the filesystem.

---

## Label System

### Type Labels (required on every PR — pick exactly one)

| Label | Use for |
|-------|---------|
| `type:bug` | Bug fixes |
| `type:feature` | New features |
| `type:docs` | Documentation-only changes |
| `type:refactor` | Code refactoring with no behavior change |
| `type:chore` | Maintenance, tooling, dependencies |
| `type:breaking-change` | Breaking changes (requires major version bump) |

### Status Labels (set by maintainers)

| Label | Meaning |
|-------|---------|
| `status:needs-review` | Awaiting maintainer review (auto-applied to new issues) |
| `status:approved` | Approved for implementation |
| `status:in-progress` | Actively being worked on |
| `status:blocked` | Blocked by another issue or external dependency |
| `status:stale` | No activity for 30 days |
| `status:wontfix` | Intentionally not fixing |

### Priority & Effort Labels (set by maintainers)

`priority:high`, `priority:medium`, `priority:low`

| Label | Meaning |
|-------|---------|
| `effort:small` | < 1 hour — good starting point for new contributors |
| `effort:medium` | 1–4 hours |
| `effort:large` | > 4 hours or spans multiple files |

---

## PR Rules

- Keep PR scope focused — one logical change per PR
- Use [conventional commits](https://www.conventionalcommits.org/) format
- Ensure all tests pass locally before pushing
- Update docs in the same PR when behavior changes

### Conventional Commit Format

```
<type>(<scope>): <short description>

[optional body]
```

**Examples:**

```
feat(cli): add --dry-run flag to sync command
fix(resolver): handle equal mtime with different hash as conflict
docs(contributing): clarify issue-first workflow
refactor(engine): extract target validation to helper
chore(deps): bump gopkg.in/yaml.v3
```

Types map to labels: `feat` → `type:feature`, `fix` → `type:bug`, `docs` → `type:docs`, `refactor` → `type:refactor`, `chore` → `type:chore`.

---

## What Gets Closed Without Merging

- PRs opened without an approved issue
- PRs that fail CI and aren't updated within 30 days
- Issues that are vague or a duplicate
- Issues with no response to a maintainer question after 14 days
