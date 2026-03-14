---
name: history
description: Search conversation history to recall past exchanges, topics, and context.
---

# History Search

Conversation history lives in `~/.ectoclaw/workspace/history/YYYY-MM-DD/messages.jsonl`.
Each line is a JSON object: `{"ts":"...","role":"user|assistant","content":"..."}`.

## When to use

- User asks about something from a past conversation
- You need context that has been compacted or forgotten
- "do you remember...", "what did I say about...", "we talked about X"

## Search

```bash
# Search across all history
grep -r "keyword" ~/.ectoclaw/workspace/history/*/messages.jsonl

# Search with jq for structured output
cat ~/.ectoclaw/workspace/history/2026-03-14/messages.jsonl \
  | jq -r 'select(.content | contains("keyword")) | "\(.ts) [\(.role)] \(.content)"'
```

If a message contains `[file: /path]`, that file still exists at that path.
