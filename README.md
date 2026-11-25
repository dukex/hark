# hark

`hark` watches your accounts and prints one line per event, so Claude Code can run it as a Monitor and react to what happens on your issues, PRs, tasks, messages and monitors.

Site: [hark.dukex.sh](https://hark.dukex.sh)

```
$ hark github
2026-11-23T10:05:00Z github issue.commented dukex/foo#3 @alice "works here" https://github.com/dukex/foo/issues/3#issuecomment-9
2026-11-23T10:08:00Z github pr.merged dukex/foo#12 @bob https://github.com/dukex/foo/pull/12
2026-11-23T10:10:00Z github pr.commented acme/lib#40 @frank watching "rebase?" https://github.com/acme/lib/pull/40#issuecomment-70
```

## Install

```
curl -fsSL https://hark.dukex.sh/install | sh
```

It downloads the latest release for your OS and architecture (Linux or macOS, amd64 or arm64), checks it against `checksums.txt` and installs it to `~/.local/bin`. Set `HARK_INSTALL_DIR` to install elsewhere, or `HARK_DOWNLOAD_URL=https://github.com/dukex/hark/releases/download/v0.1.0` to pin a version.

With Go: `go install github.com/dukex/hark/cmd/hark@latest`.

`hark --version` (or `-v`) prints the installed version.

### Checking the setup

`hark doctor` checks every provider and exits with 1 if a configured one is broken:

```
hark 0.8.0
datadog
  - not configured: set DD_API_KEY, DD_APP_KEY (...)
github
  ✓ token: logged in as you
  ✓ notifications: readable
jira
  - not configured: set JIRA_URL, JIRA_EMAIL, JIRA_API_TOKEN (...)
linear
  - not configured: set LINEAR_API_KEY (...)
slack
  ✓ token: logged in as you on acme
  ✓ search: 4 mentions and 12 DMs in the last 7 days
  ✓ state: /home/you/.local/state/hark/slack.json
todoist
  ✓ token: valid
  ✗ activity log: GET /api/v1/activities: 403 Forbidden
  ✓ state: /home/you/.local/state/hark/todoist.json
```

Providers that are not configured are listed but don't fail the check.

### Updates

On every start, a released hark checks the latest GitHub release. When it is newer, hark downloads it, verifies the checksum, replaces its own binary and restarts with the same arguments, printing `hark: updated 0.1.0 to v0.2.0, restarting` on stderr. If the check fails, hark warns and keeps running the version it has. Set `HARK_NO_UPDATE=1` to turn it off. Builds from source (`go install`, `go build`) never update themselves on start.

`hark update` updates on demand and exits, printing `hark updated 0.6.0 to v0.7.0` or `hark 0.7.0 is up to date`. It also replaces a build from source with the latest release.

## Use it from Claude Code or another agent

Paste this prompt into the session. It also works in Copilot CLI, which has no monitor but wakes the agent when a background command exits: there the agent runs hark with `--exit-on-event` and starts it again after each batch of events.

```text
Watch my Datadog, GitHub, Jira, Linear, Slack and Todoist activity with hark while we work.

1. If `hark` is not on PATH, install it with
   `curl -fsSL https://hark.dukex.sh/install | sh`.
   Then run `hark help` and read it: it describes every line hark prints,
   every flag and how to restart a monitor without losing events.
2. Run `hark doctor`. If a provider shows `✗`, tell me what failed. Only start
   monitors for providers whose checks all pass.
   Start a background monitor running `hark github 2>&1`, described as
   "GitHub activity on my issues and PRs", with the longest timeout allowed.
   When it expires, start it again. If more than an hour passed since it
   stopped, add `--since <time of its last line that is not an error>`, so
   nothing in between is lost. If your tool has no monitor but wakes you
   when a background command exits (Copilot CLI does), add `--exit-on-event`
   and run it as a detached background command instead: it exits after
   printing new events, so handle them and start it again right away, with
   `--since <time of its last line that is not an error>`. With neither, run
   it in the background and read its output every few minutes.
3. Each line is one event:
   `<time> github <kind> <repo>#<number> @<actor> [watching] "<text>" <url>`.
   - `issue.commented` / `pr.commented` without `watching`: someone commented on
     my issue or PR. Read the whole comment (`gh issue view` or `gh pr view
     --comments`), tell me what it asks for, and if it is about the work in this
     session, propose the change.
   - `pr.reviewed`, `pr.changes_requested` without `watching`: someone reviewed
     my PR. Read the review and its inline comments (`gh pr view --comments`
     and `gh api repos/<repo>/pulls/<number>/comments`) and tell me what it asks for.
   - `pr.approved`, `issue.closed`, `pr.closed`, `pr.merged`, `pr.opened`: tell me
     in one line.
   - Lines with `watching`: threads I follow but did not open. Mention them only
     if they @mention me or touch what we are working on.
   - `*.updated`: other activity, the text is GitHub's reason (for example
     `review_requested`). Tell me about review requests.
   - `ci.failed`: a workflow failed. Tell me which one and on which branch, and
     offer to read the logs (`gh run list --repo <repo> --branch <branch>`).
   - `ci.approval_requested`: someone is waiting for me to approve a deploy. Tell me.
   - `ci.succeeded`, `ci.cancelled`, `ci.skipped`: stay quiet unless I am
     waiting for that workflow.
   - `error`: hark keeps retrying and prints a repeated error only once. Tell me
     what failed.
   - A line starting with `hark:` means hark stopped. Show it to me.
4. If doctor shows jira working, start another monitor the same way with
   `hark jira 2>&1`, described as "Jira activity on my issues". Jira, Linear,
   Slack, Todoist and Datadog resume where they stopped on their own, so
   re-arm them without `--since`:
   - `jira.mentioned`, `jira.assigned`: tell me right away.
   - `jira.commented`: tell me what it asks for.
   - `jira.transitioned`: tell me in one line.
   If doctor shows linear working, start `hark linear 2>&1`, described as
   "Linear inbox":
   - `linear.mentioned`, `linear.assigned`: tell me right away.
   - `linear.commented`: tell me what it asks for.
   - `linear.transitioned`: tell me in one line.
   - Other `linear.*` kinds: stay quiet unless they touch this session's work.
   If doctor shows todoist working, start one more with `hark todoist 2>&1`,
   described as "Todoist activity":
   - `task.due`: remind me of what is due today, once.
   - `task.commented`: tell me what the comment says.
   - `task.created`, `task.completed`, `task.rescheduled`: stay quiet unless
     the task is about the work in this session.
   If doctor shows slack working, start `hark slack 2>&1`, described as
   "Slack mentions and DMs":
   - `slack.mentioned`, `slack.dm`: tell me who wrote, where, and what they
     ask for. Do not answer in Slack.
   If doctor shows datadog working, ask me which services or teams I own and
   start `hark datadog --filter '<that>' 2>&1`, described as "Datadog
   monitors": without a filter it reports every monitor in the organization.
   - `datadog.alert`, `datadog.no_data`: tell me right away, with the monitor.
   - `datadog.warn`: tell me in one line.
   - `datadog.recovered`: tell me in one line if I was told it was down.
   If I ask to watch only some repos, projects, channels, tasks or services,
   pass `--filter` with the provider's own syntax (see the hark README).
5. Never reply, react, close, mute or change anything on GitHub, Jira, Linear,
   Slack, Todoist or Datadog without asking me first.
```

## Options

`hark help` (or `hark` alone) prints the full command line reference, written for agents.

`hark <provider> [--since TIME] [--filter RULES] [--exit-on-event]`

### `--since`

`--since TIME` reports events from TIME instead of the default start below. TIME is what hark prints at the start of each line (`2026-11-23T10:05:00Z`, any RFC 3339 offset works) or a date (`2026-11-23`, midnight in your time zone). Jira, Linear, Slack, Todoist and Datadog already resume from their saved cursor, so `--since` is for going further back or skipping ahead. GitHub has no cursor and starts one hour back, so after a longer gap pass the time of the last line you saw. Use an event line: `error` and `task.due` lines carry the time hark printed them, and resuming from an `error` skips what failed to load.

### `--filter`

`--filter RULES` reports only what matches RULES, written in each provider's own syntax. `error` lines are always printed, and an invalid filter stops hark on start.

| Provider | RULES | Example |
|----------|-------|---------|
| `github` | qualifiers from the github.com notifications search: `repo:owner/name`, `org:owner`, `reason:` (`mention`, `review-requested`, `author`..., with `-` or `_`), `is:issue`, `is:pr`, `is:ci`, each negated with `-` (`-repo:acme/legacy`). Different qualifiers must all match, a repeated one matches any of its values. | `--filter 'org:acme -is:ci'` |
| `jira` | JQL, added with AND to the issues you follow | `--filter 'project = PROJ AND priority = High'`, `--filter 'project != OPS'` |
| `linear` | a Linear API `NotificationFilter` as JSON, where `type` is the notification type | `--filter '{"type":{"in":["issueMention","issueCommentMention"]}}'`, `--filter '{"type":{"nin":["issueNewComment"]}}'` |
| `slack` | Slack search modifiers, added to the mention and DM searches | `--filter 'in:#deploys'`, `--filter '-in:#random'` |
| `todoist` | a Todoist filter, as in the app | `--filter '#Work & p1'`, `--filter '!#Personal'` |
| `datadog` | Datadog event search, added to `source:alert` | `--filter 'service:checkout'`, `--filter 'team:payments -env:staging'` |

Exclude with `--filter`, not with `grep -v` after hark: GitHub notifications hark prints are marked as read, so a line dropped by a pipe disappears from your unread list unseen.

On GitHub, notifications the filter drops stay unread, and `reason:` only matches notifications, never your own actions. On Todoist, each minute with activity costs two extra requests to find the tasks that match, and `task.due` lists only tasks due today that match.

### `--exit-on-event`

`--exit-on-event` makes hark exit with status 0 after the first poll that printed an event. It is for agents that have no monitor but are woken when a background command exits, such as Copilot CLI: the agent runs `hark github --exit-on-event 2>&1` as a detached background command, goes idle, and when hark exits it handles the lines and starts hark again. While nothing happens, hark keeps polling and the agent makes no model calls.

hark exits only once the poll is done: GitHub has marked its notifications as read and the other providers have saved their cursor, so the next start picks up right after. `error` lines don't make it exit. On GitHub, pass `--since` with the time of the last event line when starting again, or your own actions made in between are skipped.

## GitHub

`hark github` uses `GITHUB_TOKEN`, or the token from `gh auth token`. Set `HARK_GITHUB_API` for GitHub Enterprise.

GitHub has no per-user webhooks, so hark polls every minute:

- `/notifications` for activity by others on issues and PRs you authored or watch. Threads authored by someone else are marked `watching`.
- `/users/{you}/events` for your own actions, which GitHub never notifies you about.

hark reads only unread notifications and marks each thread as read once it has printed it, so a restarted hark, or a second consumer, never sees the same notification twice. On startup it picks up unread notifications from the last hour, or from `--since`, which covers the gap while a monitor is re-armed; older unread ones are left untouched. Notifications already marked as read, by hark or on github.com, are not reported again even if they fall after `--since`. Threads hark does not report (releases, discussions) stay unread. Opening a thread on github.com marks it as read too, so hark skips it.

Known limits:

- GitHub delays the events API by 30 seconds up to a few hours, so your own actions can show up late.
- Each poll reports only the latest comment on a thread. Two comments within a minute show as one.
- Each poll reads one page of 50 notifications. Threads hark handles are marked as read and leave the page, so a `--since` far back takes a few polls to catch up. Threads it skips (releases, discussions, or dropped by `--filter`) stay unread and keep their place, so 50 of them hide anything older.

| Kind | When |
|------|------|
| `issue.commented`, `pr.commented` | someone comments |
| `issue.closed`, `pr.closed`, `pr.merged` | someone, or you, closes or merges |
| `pr.opened` | you open a PR |
| `pr.reviewed`, `pr.approved`, `pr.changes_requested` | someone reviews your PR, or one you watch; the text is the review body, or its first inline comment, plus the inline comment count |
| `issue.updated`, `pr.updated` | other thread activity, with the notification reason as text |
| `ci.failed`, `ci.succeeded`, `ci.cancelled`, `ci.skipped` | a workflow run you get notified about finished; the text is GitHub's title, with workflow and branch |
| `ci.approval_requested` | someone asked you to approve a deployment |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

## Jira

`hark jira` needs `JIRA_URL` (for example `https://yourcompany.atlassian.net`), `JIRA_EMAIL` and `JIRA_API_TOKEN`, from id.atlassian.com, Security, API tokens. Every minute it searches the issues you are assigned to, reported or watch, and reads their comments and history. Your own actions are skipped.

Like Todoist, it keeps its cursor in `~/.local/state/hark/jira.json` and the first run picks up the last hour. `--since` replaces the saved cursor.

| Kind | When |
|------|------|
| `jira.commented` | someone comments; the text starts with their name |
| `jira.mentioned` | a comment @mentions you |
| `jira.transitioned` | the status changes, for example `In Progress to Done` |
| `jira.assigned` | someone assigns the issue to you |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

Mentions on issues you don't follow are not reported.

## Linear

`hark linear` needs `LINEAR_API_KEY`, a personal API key from Linear, Settings, Security & access, Personal API keys. Every minute it reads your Linear inbox notifications, archived ones included, so what you clear in the app while hark is stopped is still reported.

It keeps its cursor in `~/.local/state/hark/linear.json` and the first run picks up the last hour. `--since` replaces the saved cursor.

| Kind | When |
|------|------|
| `linear.mentioned` | someone mentions you in an issue (the text is the title) or a comment (the text is the comment) |
| `linear.assigned` | someone assigns an issue to you |
| `linear.commented` | someone comments on an issue you are subscribed to |
| `linear.transitioned` | the status changes; the text ends with the status the issue has now |
| `linear.<type>` | any other notification, like `linear.unassigned_from_you` or `linear.emoji_reaction` |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

What reaches your inbox depends on your notification settings in Linear. A notification created more than 5 minutes before Linear returns it is missed.

## Slack

`hark slack` needs `SLACK_TOKEN`, a user token (`xoxp-`) with the `search:read` scope:

1. At [api.slack.com/apps](https://api.slack.com/apps), Create New App, From scratch, in your workspace.
2. In OAuth & Permissions, under User Token Scopes (not Bot Token Scopes), add `search:read`.
3. Install to Workspace (some workspaces need an admin to approve it) and copy the User OAuth Token.

Every minute it runs two Slack searches, `<@you>` for mentions and `to:me` for DMs, and reports what others wrote. Your own messages are skipped. It keeps its cursor in `~/.local/state/hark/slack.json`, the first run picks up the last hour, and `--since` replaces the saved cursor.

| Kind | When |
|------|------|
| `slack.mentioned` | someone @mentions you, in a channel or a DM |
| `slack.dm` | someone writes to you in a DM or group DM |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

Search sees a message once Slack indexes it, usually within a minute; one indexed more than 5 minutes late is missed. Slack can return only one of several matches that sit close together, and the search filters set in your Slack app apply. Threads you are in are reported only when they mention you. `hark doctor` shows how many mentions and DMs the search found in the last 7 days, so you can tell whether it sees your DMs.

## Todoist

`hark todoist` needs `TODOIST_API_TOKEN`, from Todoist settings, Integrations, Developer. It polls the activity log every minute, which depends on your Todoist plan having activity history.

hark keeps its cursor in `~/.local/state/hark/todoist.json` (or `$XDG_STATE_HOME/hark`), so a restart continues where it stopped without repeating events. The first run picks up the last hour. `--since` replaces the saved cursor.

| Kind | When |
|------|------|
| `task.created` | a task is added |
| `task.completed` | a task is completed |
| `task.rescheduled` | a task's due date changes; the text ends with the new date, or `removed` |
| `task.commented` | someone comments on a task |
| `task.due` | once a day, for each task due today |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

## Datadog

`hark datadog` needs `DD_API_KEY` and `DD_APP_KEY` (an application key that can read events), from Organization Settings, and `DD_SITE` if your site is not `datadoghq.com` (for example `datadoghq.eu` or `us5.datadoghq.com`). Every minute it reads the monitor alert events (`source:alert`) of the whole organization, so pass `--filter` with the services, teams or tags you own. A filter with a tag that does not exist is not an error, it just matches nothing.

It keeps its cursor in `~/.local/state/hark/datadog.json` and the first run picks up the last hour. `--since` replaces the saved cursor.

| Kind | When |
|------|------|
| `datadog.alert` | a monitor, or one group of a multi alert monitor, goes into Alert |
| `datadog.warn` | it goes into Warn |
| `datadog.no_data` | it stops receiving data |
| `datadog.recovered` | it goes back to OK |
| `error` | a request failed; hark keeps polling and prints the same error only once until another event arrives |

The text is the event title, the link opens the monitor. Re-notifications, and transitions that keep the same state, are not reported. Each poll looks 5 minutes back for events Datadog indexed late; anything later than that is missed.

## Adding a provider

Implement `hark.Provider` and register a constructor in `providers` in `cmd/hark/main.go`. The constructor returns an error wrapping `errNotConfigured` when credentials are missing, so `hark doctor` can tell "not configured" from "broken".

```go
type Provider interface {
	Watch(ctx context.Context, emit func(Event)) error
	Doctor(ctx context.Context) []Check
}
```
