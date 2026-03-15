<div align="center">

<img src="docs/banner.png" alt="EctoClaw" width="600">

**Turn your coding agent into a general-purpose AI assistant**

*The bridge that gives your coding agent a heartbeat*

<p>
  <img src="https://github.com/ectoclaw/ectoclaw/actions/workflows/build.yml/badge.svg" alt="CI">
  <img src="https://img.shields.io/github/v/release/ectoclaw/ectoclaw" alt="Latest release">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License">
</p>

</div>

---

Coding agents are powerful. EctoClaw makes them your personal assistant.

Just message it, like you'd message a person. It replies, remembers, and works. Ask it to find a cheap flight, contact a hotel about availability, or raise a PR on a repo you just described. It gets on with it while you do something else. It feels less like a tool and more like having someone capable on the other end of the chat.

## Quick Start

**1. Install**

```bash
curl -sSL ectoclaw.com/install.sh | sh
```

Or download a binary for your platform from the [releases page](https://github.com/ectoclaw/ectoclaw/releases) and move it to `/usr/local/bin/`.

**2. Configure**

```bash
ectoclaw onboard
```

This creates `~/.ectoclaw/config.json` and your workspace. Open the config and add your channel token (Telegram bot token, Discord token, etc.) and the path to your coding agent binary.

> **Supported agents:** [Claude Code](https://claude.ai/download) (`claude`) and [OpenAI Codex](https://github.com/openai/codex) (`codex`). Set `bridge.provider` in your config or via `ECTOCLAW_BRIDGE_PROVIDER`.

**3. Run**

As a one-off (useful for testing your setup):
```bash
ectoclaw gateway
```

As a persistent background service:
```bash
sudo ectoclaw service install
sudo ectoclaw service start
ectoclaw service status
```

## Yet Another Claw?

There are plenty of tools that connect chat platforms to AI. Most of them manage their own tools, their own memory, their own skill systems — and route through pay-per-token LLM APIs, adding a separate bill on top of whatever you're already paying.

EctoClaw doesn't do any of that. Instead of reinventing what your coding agent already does well, it just gets out of the way: it hands the message to `claude` or `codex`, lets the agent do its thing, and sends the response back to your chat. Same binary you run at your desk. Same tools, sessions, and MCP servers. No extra keys. No extra bill.

The difference is respect: other tools treat coding agents as dumb LLM endpoints. EctoClaw treats them as capable agents and lets them prove it.

What EctoClaw adds on top of the bridge:

- **Conversation history** — every chat has its own agent session. Ask "what did we talk about last Tuesday?" and it actually knows.
- **Proactive heartbeat** — the agent can reach out to *you*. Morning briefings, reminders, weekly summaries — all scheduled, all delivered to your chat.
- **Assistant persona** — `SOUL.md`, `IDENTITY.md`, and `MEMORY.md` give the agent a name, personality, and memory about you. It behaves like someone you've hired, not a tool you're prompting.

## What It Supports

- **Channels** — Telegram, WhatsApp, Discord, Slack, IRC, Matrix, LINE
- **Providers** — Claude Code (`claude`), OpenAI Codex (`codex`)
- **Conversation history** — remembers everything you've discussed, no matter how much time has passed
- **Cron & scheduled tasks** — jobs that run on a schedule and message you back
- **Heartbeat** — proactive check-ins on a configurable interval
- **Workspace memory** — `SOUL.md`, `IDENTITY.md`, `MEMORY.md`, daily logs
- **Skills** — extend the agent with skills: `cron`, `history`, `weather`, and more

## Usage

Talk to your assistant from any connected chat:

```
@claw what's the weather in Berlin this week?
@claw summarize my git log from the past 7 days
@claw every weekday at 9am, send me the top HN posts
@claw open a PR for the current branch
```

Manage it from your private chat:

```
@claw list all scheduled tasks
@claw pause the morning briefing
@claw what sessions are active?
```

## Customizing

EctoClaw reads your personality and context from `~/.ectoclaw/workspace/`. These files are concatenated into the system prompt for every conversation.

| File | What goes here |
|------|----------------|
| `BOOTSTRAP.md` | First-run setup script. When present, the agent asks you a few questions, fills in your name, timezone, and agent name, then deletes the file. Created automatically by `ectoclaw onboard`. |
| `SYSTEM.md` | Behavioural rules for operating inside a chat app — when to act vs ask, how to handle long tasks, how to write for small screens. |
| `SOUL.md` | Your agent's personality — tone, values, how it talks and makes decisions. This is the one file you're expected to author yourself. |
| `IDENTITY.md` | Your agent's identity: name, creature type, vibe, and emoji. Filled in during first-run setup. |
| `USER.md` | Information about you: your name, timezone, background, and how you like responses. The agent uses this to tailor replies to you specifically. |
| `MEMORY.md` | Facts worth keeping across conversations — preferences, recurring needs, things the agent has learned about you. The agent writes here; you can edit or clear it any time. |
| `HEARTBEAT.md` | Tasks the agent checks on every heartbeat tick — weather alerts, reminders, GitHub notifications, etc. |
| `history/YYYY-MM-DD/messages.jsonl` | Daily conversation logs. The two most recent days are loaded each session so the agent remembers what you were working on. |

You can tell the agent directly to update any of these:

```
"Remember that I prefer concise replies"
"Update my USER.md — I'm now based in Berlin"
"Add a skill that summarizes yesterday's commits"
```

## Contributing

**Add skills, not features.** If you want new behaviour, create a skill in `workspace/skills/` rather than touching the core. Open a PR so others can pull it into their setup.

Bug fixes and clear improvements to the bridge are always welcome.

## License

MIT
