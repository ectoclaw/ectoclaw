# Contributing to EctoClaw

Thank you for your interest in contributing to EctoClaw! We welcome contributions of all kinds: bug fixes, features, documentation, and testing. Be kind, constructive, and assume good faith — harassment or discrimination of any kind will not be tolerated.

EctoClaw itself was substantially developed with AI assistance — we embrace this approach and have built our contribution process around it.

---

## Project Background

EctoClaw is a thin bridge between chat channels and coding-agent CLIs. It delegates AI work entirely to the `claude` or `codex` subprocess — no direct API calls, no extra billing. The codebase is intentionally small and focused: bridging channels, managing sessions, assembling prompts, and scheduling jobs.

For substantial new features, open an issue first to discuss whether they fit the project's scope.

---

## Getting Started

### Prerequisites

- Go 1.25 or later
- `make`
- `claude` CLI installed and authenticated (`claude auth login`), or `codex` CLI if using the codex provider

### Fork and Clone

```bash
git clone https://github.com/<your-username>/ectoclaw.git
cd ectoclaw
git remote add upstream https://github.com/ectoclaw/ectoclaw.git
```

### Build and Test

```bash
make build    # Build binary (runs go generate first)
make check    # Full pre-commit check: deps + fmt + vet + test
make test     # Run tests only
make fmt      # Format code
make lint     # Full linter run
```

Run `make check` locally before pushing — it runs the same checks as CI.

---

## Contributing

### What You Can Contribute

- **Bug reports** — Open an issue using the bug report template.
- **Feature requests** — Open an issue using the feature request template. Discuss before implementing.
- **Code** — Fix bugs or implement features following the workflow below.
- **Documentation** — Improve READMEs, docs, or inline comments.
- **Testing** — Run EctoClaw on new hardware, channels, or Claude CLI versions and report your results.

### Branches and Commits

Branch off `main` and target `main` in your PR:

```bash
git checkout main && git pull upstream main
git checkout -b your-feature-branch
```

Use descriptive branch names: `fix/telegram-timeout`, `feat/irc-channel`, `docs/setup-guide`.

Write clear commit messages in English using the imperative mood: `Add retry logic`, not `Added retry logic`. Reference issues where relevant: `Fix session leak (#123)`. Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Rebase onto upstream `main` before opening a PR:

```bash
git fetch upstream && git rebase upstream/main
```

### Opening a Pull Request

Fill in the PR template completely. Keep PRs focused and small — a 200-line PR across 5 files is much easier to review than a 2000-line PR across 30 files. If your feature is large, split it into a series of smaller logically complete PRs.

The PR template asks for: a description of what and why, type of change, related issue, test environment (OS, Claude version, channels), and optional evidence (logs or screenshots).

---

## AI-Assisted Development

EctoClaw was built with substantial AI assistance, and we fully embrace AI-assisted development. However, contributors are fully responsible for what they submit.

Before opening a PR with AI-generated code, you must:

- **Read and understand** every line of the generated code.
- **Test it** in a real environment — fill in the test environment section of the PR template.
- **Check for security issues** — AI models can produce subtly insecure code: path traversal, injection, credential exposure. Review carefully.
- **Verify correctness** — AI-generated logic can be plausible-sounding but wrong. Validate behavior, not just syntax.

PRs where it is clear the contributor has not read or tested the AI-generated code will be closed without review.

AI-generated contributions are held to the **same quality bar** as human-written code: passing CI, idiomatic Go, consistent with the existing style, no unnecessary abstractions or dead code, and tests where appropriate.

Pay extra security attention to: file path handling, input validation in channel handlers, credential handling, and command execution (`exec.Command`, shell invocations). If you are unsure whether a piece of code is safe, say so in the PR — reviewers will help.

---

## Review and Merge

### Merging into `main`

A PR can be merged when: CI is green, at least one maintainer has approved, all review threads are resolved, and the PR template is complete. Only maintainers can merge PRs.

We use **squash merge** to keep `main` history clean. Each merged PR becomes a single commit, e.g. `feat: Add Matrix placeholder support (#42)`. A regular merge may be used at the maintainer's discretion when the commits tell a clear independent story.

### Code Review

When reviewing, focus on: correctness and edge cases, security (especially for AI-generated code and channel handlers), architectural consistency, simplicity, and test coverage.

Be constructive and specific. "This could have a race condition if two goroutines call this concurrently — consider using a mutex" is better than "this looks wrong".

As a contributor, respond to review comments in a reasonable time. Do not force-push after a review has started — use additional commits instead; the maintainer will squash on merge. If you disagree with feedback, engage respectfully.

### Release Branches

When a version is ready, maintainers cut a `release/x.y` branch from `main`. New features are not backported — the release branch only receives security fixes and critical bug fixes via cherry-pick. If you believe a fix qualifies, note it in the PR or open a separate issue.

### Communication

- **GitHub Issues** — Bug reports, feature requests, design discussions.
- **GitHub Discussions** — General questions, ideas, community conversation.
- **Pull Request comments** — Code-specific feedback.

When in doubt, open an issue before writing code. It costs little and prevents wasted effort.

---

EctoClaw was substantially designed and implemented with AI assistance, guided by human oversight. If you find something that looks odd, over-engineered, or out of scope — opening an issue is always welcome.
