# hark

`hark` watches your accounts and prints one line per event, so Claude Code can run it as a Monitor and react to what happens on your issues and PRs.

```
$ hark github
2026-11-23T10:05:00Z github issue.commented dukex/foo#3 @alice "works here" https://github.com/dukex/foo/issues/3#issuecomment-9
2026-11-23T10:08:00Z github pr.merged dukex/foo#12 @bob https://github.com/dukex/foo/pull/12
2026-11-23T10:10:00Z github pr.commented acme/lib#40 @frank watching "rebase?" https://github.com/acme/lib/pull/40#issuecomment-70
```

## Install

```
curl -fsSL https://raw.githubusercontent.com/dukex/hark/main/install.sh | sh
```

It downloads the latest release for your OS and architecture (Linux or macOS, amd64 or arm64), checks it against `checksums.txt` and installs it to `~/.local/bin`. Set `HARK_INSTALL_DIR` to install elsewhere, or `HARK_DOWNLOAD_URL=https://github.com/dukex/hark/releases/download/v0.1.0` to pin a version.

With Go: `go install github.com/dukex/hark/cmd/hark@latest`.

`hark --version` (or `-v`) prints the installed version.

### Checking the setup

`hark doctor` checks every provider and exits with 1 if a configured one is broken:

```
hark 0.8.0
github
  ✓ token: logged in as you
  ✓ notifications: readable
jira
  - not configured: set JIRA_URL, JIRA_EMAIL, JIRA_API_TOKEN (...)
todoist
  ✓ token: valid
  ✗ activity log: GET /api/v1/activities: 403 Forbidden
```

Providers that are not configured are listed but don't fail the check.

### Updates

On every start, a released hark checks the latest GitHub release. When it is newer, hark downloads it, verifies the checksum, replaces its own binary and restarts with the same arguments, printing `hark: updated 0.1.0 to v0.2.0, restarting` on stderr. If the check fails, hark warns and keeps running the version it has. Set `HARK_NO_UPDATE=1` to turn it off. Builds from source (`go install`, `go build`) never update themselves.

## Use it from Claude Code or another agent

Paste this prompt into the session:

```text
Watch my GitHub, Jira and Todoist activity with hark while we work.

1. If `hark` is not on PATH, install it with
   `curl -fsSL https://raw.githubusercontent.com/dukex/hark/main/install.sh | sh`.
2. Run `hark doctor`. If a provider shows `✗`, tell me what failed. Only start
   monitors for providers whose checks all pass.
   Start a background monitor running `hark github 2>&1`, described as
   "GitHub activity on my issues and PRs", with the longest timeout allowed.
   When it expires, start it again. If your tool has no monitor, run it in
   the background and read its output every few minutes.
3. Each line is one event:
   `<time> github <kind> <repo>#<number> @<actor> [watching] "<text>" <url>`.
   - `issue.commented` / `pr.commented` without `watching`: someone commented on
     my issue or PR. Read the whole comment (`gh issue view` or `gh pr view
     --comments`), tell me what it asks for, and if it is about the work in this
     session, propose the change.
   - `issue.closed`, `pr.closed`, `pr.merged`, `pr.opened`: tell me in one line.
   - Lines with `watching`: threads I follow but did not open. Mention them only
     if they @mention me or touch what we are working on.
   - `*.updated`: other activity, the text is GitHub's reason (for example
     `review_requested`). Tell me about review requests.
   - `ci.failed`: a workflow failed. Tell me which one and on which branch, and
     offer to read the logs (`gh run list --repo <repo> --branch <branch>`).
   - `ci.approval_requested`: someone is waiting for me to approve a deploy. Tell me.
   - `ci.succeeded`, `ci.cancelled`, `ci.skipped`: stay quiet unless I am
     waiting for that workflow.
   - `error`: hark keeps retrying. Tell me only if it repeats.
   - A line starting with `hark:` means hark stopped. Show it to me.
4. If doctor shows jira working, start another monitor the same way with
   `hark jira 2>&1`, described as "Jira activity on my issues":
   - `jira.mentioned`, `jira.assigned`: tell me right away.
   - `jira.commented`: tell me what it asks for.
   - `jira.transitioned`: tell me in one line.
   If doctor shows todoist working, start one more with `hark todoist 2>&1`,
   described as "Todoist activity":
   - `task.due`: remind me of what is due today, once.
   - `task.commented`: tell me what the comment says.
   - `task.created`, `task.completed`, `task.rescheduled`: stay quiet unless
     the task is about the work in this session.
5. Never reply, close or change anything on GitHub, Jira or Todoist without
   asking me first.
```

## GitHub

`hark github` uses `GITHUB_TOKEN`, or the token from `gh auth token`. Set `HARK_GITHUB_API` for GitHub Enterprise.

GitHub has no per-user webhooks, so hark polls every minute:

- `/notifications` for activity by others on issues and PRs you authored or watch. Threads authored by someone else are marked `watching`.
- `/users/{you}/events` for your own actions, which GitHub never notifies you about.

hark reads only unread notifications and marks each thread as read once it has printed it, so a restarted hark, or a second consumer, never sees the same notification twice. On startup it picks up unread notifications from the last hour, which covers the gap while a monitor is re-armed; older unread ones are left untouched. Threads hark does not report (releases, discussions) stay unread. Opening a thread on github.com marks it as read too, so hark skips it.

Known limits:

- GitHub delays the events API by 30 seconds up to a few hours, so your own actions can show up late.
- Each poll reports only the latest comment on a thread. Two comments within a minute show as one.

| Kind | When |
|------|------|
| `issue.commented`, `pr.commented` | someone comments |
| `issue.closed`, `pr.closed`, `pr.merged` | someone, or you, closes or merges |
| `pr.opened` | you open a PR |
| `issue.updated`, `pr.updated` | other thread activity, with the notification reason as text |
| `ci.failed`, `ci.succeeded`, `ci.cancelled`, `ci.skipped` | a workflow run you get notified about finished; the text is GitHub's title, with workflow and branch |
| `ci.approval_requested` | someone asked you to approve a deployment |
| `error` | a request failed; hark keeps polling |

## Jira

`hark jira` needs `JIRA_URL` (for example `https://yourcompany.atlassian.net`), `JIRA_EMAIL` and `JIRA_API_TOKEN`, from id.atlassian.com, Security, API tokens. Every minute it searches the issues you are assigned to, reported or watch, and reads their comments and history. Your own actions are skipped.

Like Todoist, it keeps its cursor in `~/.local/state/hark/jira.json` and the first run picks up the last hour.

| Kind | When |
|------|------|
| `jira.commented` | someone comments; the text starts with their name |
| `jira.mentioned` | a comment @mentions you |
| `jira.transitioned` | the status changes, for example `In Progress to Done` |
| `jira.assigned` | someone assigns the issue to you |
| `error` | a request failed; hark keeps polling |

Mentions on issues you don't follow are not reported.

## Todoist

`hark todoist` needs `TODOIST_API_TOKEN`, from Todoist settings, Integrations, Developer. It polls the activity log every minute, which depends on your Todoist plan having activity history.

hark keeps its cursor in `~/.local/state/hark/todoist.json` (or `$XDG_STATE_HOME/hark`), so a restart continues where it stopped without repeating events. The first run picks up the last hour.

| Kind | When |
|------|------|
| `task.created` | a task is added |
| `task.completed` | a task is completed |
| `task.rescheduled` | a task's due date changes; the text ends with the new date, or `removed` |
| `task.commented` | someone comments on a task |
| `task.due` | once a day, for each task due today |
| `error` | a request failed; hark keeps polling |

## Adding a provider

Implement `hark.Provider` and register a constructor in `providers` in `cmd/hark/main.go`. The constructor returns an error wrapping `errNotConfigured` when credentials are missing, so `hark doctor` can tell "not configured" from "broken".

```go
type Provider interface {
	Watch(ctx context.Context, emit func(Event)) error
	Doctor(ctx context.Context) []Check
}
```
