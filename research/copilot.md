# hark on GitHub Copilot CLI

Status on 2026-11-25: **could work today with one small hark option, not tested yet.** Every claim below comes from Copilot CLI's docs and issue tracker. Nothing has been run against Copilot CLI.

## Same gap as Codex

Copilot CLI has no equivalent of Claude Code's Monitor. To wait on a background process the agent calls `read_bash` in a loop, and each check is a model turn. See [codex.md](codex.md) for why that matters to hark.

[github/copilot-cli#5260](https://github.com/github/copilot-cli/issues/5260) asks for a `monitor` tool, or a `notify_on` option on async `bash`, that wakes the agent on each line a background process prints. It was opened on 2026-10-26, labelled `area:tools`, and as of 2026-11-25 it has no comments, no reactions and no reply from GitHub.

## What Copilot CLI has that Codex doesn't: a wake when a process exits

- Async `bash` sends a completion notification when the background process exits ([#5260](https://github.com/github/copilot-cli/issues/5260)).
- With `detach: true`, the completion wake still runs: notification, then `read_bash`, then a new turn ([#5293](https://github.com/github/copilot-cli/issues/5293), [t3code#27301](https://github.com/pingdotgg/t3code/issues/27301)).
- In Autopilot, a detached shell that finishes after `task_complete` has its notification held until the user's next message. 1.0.106 reportedly stopped those notifications from being dropped, but not necessarily from being held ([#5430](https://github.com/github/copilot-cli/issues/5430)).
- The [hooks reference](https://docs.github.com/en/copilot/reference/hooks-reference) lists a `shell_completed` event for async shells, in Copilot CLI only.

Codex lacks this: there, background processes don't wake the agent when they finish (openai/codex#15723, cited in openai/codex#29922).

## Idea: exit after events, and let the exit wake the agent

Give hark an option, maybe `--exit-on-event`, that makes it stop after the first poll that printed an event.

The agent starts `hark github --exit-on-event 2>&1` as a detached async shell and ends its turn. While nothing happens, hark keeps polling and the agent makes no model calls. When events arrive, hark prints them and exits. The completion notification wakes the agent, which reads the lines, handles them the way the prompt says, and starts hark again.

That is a Monitor with one restart per batch of events. Nothing is lost between exit and restart, as long as hark exits only after the poll's state is saved:

- GitHub marks each notification as read inside the poll, before printing it. Unread ones stay unread, and the next start picks them up (one hour back by default, or `--since`).
- Datadog, Jira, Slack and Todoist save their cursor right after printing a poll's events (`hark.SaveState` after the `emit` loop in each `Watch`). Exiting after that save means the restart continues exactly there.

Open questions for the design:

- Should an `error` line make hark exit? Waking the agent for every transient failure is noisy, but never exiting on a persistent one hides it.
- The name, and whether it belongs in `help.txt`'s FOR AGENTS section as the way to run hark without a monitor.
- Nothing in it is specific to Copilot CLI. It fits any agent that wakes when a background command exits.

## To validate

Needs Copilot CLI installed and logged in.

- [ ] **Wake on exit, no hark.** Ask the agent to start `sh -c 'sleep 90; echo "2026-11-25T10:05:00Z github pr.changes_requested acme/x#1 @carol \"split it\""'` as a detached async shell and end its turn. Does the idle session wake on its own when it exits, and does it read the line?
- [ ] **Same in Autopilot**, after `task_complete` ([#5430](https://github.com/github/copilot-cli/issues/5430)).
- [ ] If it wakes: add the exit-after-events option to hark with tests, then document it in `help.txt`, README (a "Use it from Copilot CLI" prompt) and the site's setup and FAQ.
- [ ] Revisit [#5260](https://github.com/github/copilot-cli/issues/5260). If a native monitor ships, Copilot CLI can run hark the way Claude Code does.
