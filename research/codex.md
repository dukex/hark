# hark on Codex

Status on 2026-11-25: **waiting for upstream.** Nothing here has been tested against a running Codex yet; every claim below comes from Codex's docs, its issue tracker or third-party write-ups.

## Why Codex is different

hark works in Claude Code because its Monitor tool delivers each line hark prints to the agent as it arrives, and wakes an idle session to handle it. The agent sits idle at no cost and reacts within a minute.

Codex CLI (0.169 at the time of writing) has no equivalent. It is turn-driven: between turns it cannot react to anything, and an agent that runs hark in a background terminal only sees new lines when it decides to look. That is the polling hark is meant to replace.

## The upstream fix: a `monitor` tool

[openai/codex#29922](https://github.com/openai/codex/issues/29922) proposes an agent-callable `monitor(action="start", command=...)` tool in codex-core: each line the command prints wakes the idle session, with no model calls while it is quiet. It names Claude Code's Monitor as the model.

If it ships, hark needs no code change. The prompt in README would say to start a monitor running `hark <provider> 2>&1`, the same as in Claude Code.

Where it stands:

- Opened 2026-06-24, labelled `enhancement`. No reply from an OpenAI maintainer as of 2026-11-25, despite requests for triage in October and November.
- Working reference implementations exist: a fork on v0.142 by the issue's author, and [a community fork on 0.169](https://github.com/chriswangcq/codex/tree/codex/monitor-preview).
- Related and also open: [#22003](https://github.com/openai/codex/issues/22003) (inject background command output into the active session) and [#20312](https://github.com/openai/codex/issues/20312) (event-driven session wake).

**Decision:** wait for #29922 for the real integration. Don't build Codex-specific machinery in hark that a native monitor would make obsolete.

## What Codex has today

| Mechanism | Reaches a working agent | Wakes an idle session | Confidence |
|---|---|---|---|
| [Hooks](https://developers.openai.com/codex/hooks) (stable since 0.124) | Yes: `PostToolUse` can add `additionalContext` after each tool call, and `Stop` with `decision: "block"` turns its `reason` into a new prompt when a turn ends | No. The docs say finishing a hook never starts a turn; background (`async`) hook output waits for the next user turn | Documented |
| [`codex queue --thread <id> --message "..."`](https://developers.openai.com/codex/cli/slash-commands) (0.149+, `--resume` in 0.166) | Waits for the running turn to end | Probably. [codex-long-jobs](https://github.com/zycdev/codex-long-jobs) relies on it to wake the original conversation, but the official docs don't say | Unverified. [#56914](https://github.com/openai/codex/issues/56914) and [#52705](https://github.com/openai/codex/issues/52705) report queued messages stuck, and the command exits 0 once the message is stored, delivered or not |
| [App server](https://developers.openai.com/codex/app-server): `turn/start`, `turn/steer`, `thread/inject_items` | `turn/steer` adds input to the running turn | `turn/start` might | Undocumented whether a second client can attach to the TUI's thread and whether the TUI shows the turn. `codex queue` is the supported wrapper around this (`thread/queue/add`) |
| Background terminals (`unified_exec`, `/ps`, `/stop`) | Only when the model polls | No | Documented |
| [MCP events](https://developers.openai.com/plugins/build/mcp-events) | An MCP server can return an event later without polling | Not established | Experimental, described as limited in #29922 |

Making hark exit after each batch of events, so the agent wakes when the process ends, works in Copilot CLI (see [copilot.md](copilot.md)) but not here: Codex background processes don't wake the agent when they finish (openai/codex#15723).

## Interim bridge, if Codex users need hark before #29922

Run hark outside the agent and push its lines into the session with `codex queue`, batching lines that arrive within a few seconds so each burst is one turn, not one turn per line.

Caveat: hark marks every GitHub notification it prints as read. If `codex queue` stores a message that is never delivered, that event is gone from the user's unread list without anyone seeing it. Any bridge has to write lines to a file first and only treat them as delivered once Codex has them.

## To validate

Needs Codex installed and logged in (`npm i -g @openai/codex`, `codex login`).

- [ ] **Idle wake.** Open Codex, `/rename hark-test`, then from another shell run `codex queue --thread hark-test --message '2026-11-25T10:05:00Z github pr.changes_requested acme/x#1 @carol "split it"'`. Does the idle session start a turn on its own?
- [ ] **During a turn.** Same, while the agent is working. Is the message delivered after the turn, as a user message, and shown in the TUI?
- [ ] **Thread id from inside the session.** Can the agent find its own thread id (`/status`, an environment variable, the session name), so it can start the bridge itself?
- [ ] **Daemon modes.** Does it still work when the TUI runs with `--no-daemon`, and after `codex resume` (see #56914)?
- [ ] If the above pass: a "Use it from Codex" section in README with the bridge, and the same in the site's setup.
- [ ] Revisit #29922. If it shipped, test `monitor` with `hark github 2>&1`, update the prompt for Codex and drop the bridge.

## Already true without Codex

The site's FAQ ("Which agents does hark work with?") says other agents can use hark "the same way". Until a monitor exists in those agents, they only see events when they check. The answer should say so.
