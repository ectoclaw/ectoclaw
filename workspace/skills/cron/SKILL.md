---
name: cron
description: Schedule, list, and manage recurring jobs via the ectoclaw cron command.
---

# Cron

Manage scheduled tasks with the `ectoclaw cron` command.

## Add a job

Recurring cron expression (standard 5-field format):
```bash
ectoclaw cron add --name "Morning reminder" --cron "30 9 * * *" \
  --message "Time to drink water!" --deliver \
  --channel telegram --to <chat_id>
```

Run every N seconds (simpler for intervals):
```bash
ectoclaw cron add --name "Hourly check" --every 3600 \
  --message "Check for new emails" \
  --channel telegram --to <chat_id>
```

Flags:
- `--name` (required): Human-readable job name
- `--message` (required): Message delivered or sent to Claude
- `--cron`: Standard cron expression (`"30 9 * * *"` = 9:30 daily)
- `--every`: Interval in seconds (mutually exclusive with `--cron`)
- `--deliver`: Send message directly to user without invoking Claude
- `--channel`: Channel name (e.g. `telegram`)
- `--to`: Chat ID / recipient

## List jobs
```bash
ectoclaw cron list
```

## Remove a job
```bash
ectoclaw cron remove <job-id>
```

## Enable / disable a job
```bash
ectoclaw cron enable <job-id>
ectoclaw cron disable <job-id>
```

## Notes

- Without `--deliver`, the message is sent to Claude as a prompt — Claude generates a response and delivers it to the channel.
- With `--deliver`, the message is delivered as-is (no Claude invocation).
- `--channel` and `--to` are required when `--deliver` is set.
- To find your chat ID: send any message to the bot, check the gateway logs for `chat_id`.
