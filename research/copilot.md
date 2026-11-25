# hark on GitHub Copilot CLI

Status on 2026-11-25: **works with `hark <provider> --exit-on-event`**, added in PR #33. Tested end to end with Copilot CLI 1.0.86 over ACP. Copilot's interactive TUI and Autopilot are not tested yet.

## Same gap as Codex

Copilot CLI has no equivalent of Claude Code's Monitor. To wait on a background process the agent calls `read_bash` in a loop, and each check is a model turn. See [codex.md](codex.md) for why that matters to hark.

[github/copilot-cli#5260](https://github.com/github/copilot-cli/issues/5260) asks for a `monitor` tool, or a `notify_on` option on async `bash`, that wakes the agent on each line a background process prints. It was opened on 2026-10-26, labelled `area:tools`, and as of 2026-11-25 it has no comments, no reactions and no reply from GitHub.

## What Copilot CLI has that Codex doesn't: a wake when a process exits

- Async `bash` sends a completion notification when the background process exits ([#5260](https://github.com/github/copilot-cli/issues/5260)).
- With `detach: true`, the completion wake still runs: notification, then `read_bash`, then a new turn ([#5293](https://github.com/github/copilot-cli/issues/5293), [t3code#27301](https://github.com/pingdotgg/t3code/issues/27301)).
- In Autopilot, a detached shell that finishes after `task_complete` has its notification held until the user's next message. 1.0.106 reportedly stopped those notifications from being dropped, but not necessarily from being held ([#5430](https://github.com/github/copilot-cli/issues/5430)).
- The [hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference) lists a `shell_completed` event for async shells, in Copilot CLI only.

Codex lacks this: there, background processes don't wake the agent when they finish (openai/codex#15723, cited in openai/codex#29922).

## How it works: exit after events, and let the exit wake the agent

`--exit-on-event` makes hark stop after the first poll that printed an event.

The agent starts `hark github --exit-on-event 2>&1` as a detached async shell and ends its turn. While nothing happens, hark keeps polling and the agent makes no model calls. When events arrive, hark prints them and exits. The completion notification wakes the agent, which reads the lines, handles them the way the prompt says, and starts hark again.

That is a Monitor with one restart per batch of events. Nothing is lost between exit and restart, as long as hark exits only after the poll's state is saved:

- GitHub marks each notification as read inside the poll, before printing it. Unread ones stay unread, and the next start picks them up (one hour back by default, or `--since`).
- Datadog, Jira, Slack and Todoist save their cursor right after printing a poll's events (`hark.SaveState` after the `emit` loop in each `Watch`). Exiting after that save means the restart continues exactly there.

Decisions:

- `error` lines don't make hark exit. Each restart forgets which errors it already printed, so exiting on errors would wake the agent every minute while a service is down.
- `help.txt` lists it in FOR AGENTS as the way to run hark without a monitor, and the README prompt tells such agents to use it. Nothing in it is specific to Copilot CLI: it fits any agent that wakes when a background command exits.
- GitHub keeps no cursor for your own actions (`pr.opened`, closing, merging). Ones made between the exit and the next start are only reported if the next start passes `--since` with the time of the last event line, so the prompt says to.

## Validation

Run with Copilot CLI 1.0.86 started as `copilot --acp --allow-all-tools`, driving it over the Agent Client Protocol: one `session/prompt`, then listening without sending anything else.

- [x] **Wake on exit, no hark.** The agent started `sh -c 'sleep 25; echo ...'` as a detached shell and ended its turn (`stopReason: end_turn` at 28s). The shell exited at about 48s, a new turn started at 50s without any input, and at 56s the agent replied with the line.
- [x] **End to end with hark.** The agent started a build of `hark todoist --exit-on-event` against a fake Todoist API that returns an event after 20 seconds, and ended its turn at 25s. hark's second poll, at 79s, printed the event, saved the cursor and exited. Copilot woke the agent within a second, and it replied with the line. `todoist.json` held the cursor at the event.
- [ ] **Interactive TUI.** Expected to match ACP, since both run the same agent core, but not run.
- [ ] **Autopilot**, after `task_complete` ([#5430](https://github.com/github/copilot-cli/issues/5430)). The notification may be held until your next message.
- [ ] Revisit [#5260](https://github.com/github/copilot-cli/issues/5260). If a native monitor ships, Copilot CLI can run hark the way Claude Code does, without restarts.
