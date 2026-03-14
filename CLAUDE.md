# EctoClaw

EctoClaw is a thin Go bridge that connects chat platforms (Telegram, Discord, WhatsApp, Slack, Matrix, LINE, IRC) to the `claude` CLI. It is **not** an orchestrator — it invokes the `claude` binary as a subprocess and forwards responses back through your chat channel of choice.

## Tech Stack

- **Language:** Go 1.25+ with `CGO_ENABLED=0` (fully static binaries)
- **CLI framework:** `github.com/spf13/cobra`
- **Chat SDKs:** telego (Telegram), discordgo, whatsmeow, slack-go, mautrix, irc-go
- **AI:** anthropic-sdk-go for Anthropic API; `claude` subprocess for Claude Code mode
- **Scheduling:** `github.com/adhocore/gronx` — standard 5-field cron expressions
- **SQLite:** `modernc.org/sqlite` (pure Go, no CGO required)
- **Build tag:** `-tags stdjson` uses stdlib JSON instead of sonic

## Project Layout

```
cmd/ectoclaw/          CLI entry point and subcommands (agent, gateway, cron, skills, onboard, status)
pkg/bridge/            Core engine: invokes claude subprocess, manages sessions, assembles prompts
pkg/channels/          Platform adapters — one subdirectory per chat platform
pkg/bus/               In-process message bus (inbound/outbound channels)
pkg/config/            Config loading, defaults, migration
pkg/cron/              Cron job service (gronx-based)
pkg/heartbeat/         Proactive check-in service
pkg/skills/            Skills loader, ClawHub registry integration
pkg/state/             Session state persistence
workspace/             Default workspace template embedded into the binary at build time
```

### Bridge internals (`pkg/bridge/`)

| File | Purpose |
|------|---------|
| `invoke.go` | Runs `claude` as a subprocess, parses `stream-json` output |
| `loop.go` | Main event loop: consume inbound → invoke → publish outbound |
| `sessions.go` | Persists `SessionKey → claude_session_id` in `sessions.json` |
| `history.go` | Appends exchanges to daily JSONL files in `workspace/history/` |
| `prompt.go` | Assembles system prompt from `SOUL.md`, `IDENTITY.md`, `USER.md`, `MEMORY.md`, recent daily logs |
| `output.go` | Extracts result text from `claude` JSON output |

### Message flow

```
Channel adapter → bus.inbound → Loop.handleMessage() → Invoke(claude) → bus.outbound → Channel adapter
```

Each chat has a `SessionKey` (e.g., `telegram:chat_123`). Sessions are resumed across restarts using the Claude Code `--resume` flag.

## Build & Test

```bash
make build          # Build for current platform
make build-all      # Build for all supported platforms
make install        # Install to ~/.local/bin

make check          # Full pre-commit check: deps + fmt + vet + test
make test           # Run tests only
make fmt            # Format code
make lint           # Run golangci-lint
make fix            # Auto-fix lint issues

make docker-build   # Build minimal Alpine image
make docker-run     # Run gateway in Docker
```

Always run `make check` before committing. The CI pipeline runs the same steps.

## Runtime Commands

```bash
ectoclaw onboard    # Initialize config & workspace (~/.ectoclaw/)
ectoclaw gateway    # Start long-running bot (main production mode)
ectoclaw agent -m "hello"   # One-shot agent interaction
ectoclaw cron       # Manage scheduled jobs
ectoclaw skills     # Install/remove/list skills
ectoclaw status     # Show active sessions and system info
```

## Configuration

Config lives at `~/.ectoclaw/config.json`. See `config/config.example.json` for the full structure. Key sections: `bridge` (workspace path, model), `channels` (per-platform tokens and `allow_from` lists), `heartbeat`, `cron`, `skills`, `voice`.

## Workspace

The workspace (`~/.ectoclaw/workspace/`) is embedded at build time from the `workspace/` directory in this repo. It provides the system prompt context for every Claude invocation. Users customize `SOUL.md`, `IDENTITY.md`, `USER.md`, and `MEMORY.md`. Skills live in `workspace/skills/<name>/SKILL.md`.

## Key Conventions

- **No CGO.** Keep all dependencies pure-Go or static. New dependencies that require CGO are not acceptable.
- **Line length:** 120 characters (enforced by golangci-lint).
- **Function length:** max 120 lines / 40 statements (enforced by linter).
- **Atomic writes** for any file that persists state (sessions, history). See `pkg/fileutil/file.go`.
- **Channel adapters** must implement the interface in `pkg/channels/interfaces.go`. Use `base.go` for shared logic.
- Cron expressions use 5 fields (no seconds field).
