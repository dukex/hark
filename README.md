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

## Use it from Claude Code or another agent

Paste this prompt into the session:

```text
Watch my GitHub activity with hark while we work.

1. If `hark` is not on PATH, install it with
   `curl -fsSL https://raw.githubusercontent.com/dukex/hark/main/install.sh | sh`.
2. Start a background monitor running `hark github 2>&1`, described as
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
   - `error`: hark keeps retrying. Tell me only if it repeats.
   - A line starting with `hark:` means hark stopped. Show it to me.
4. Never reply, close or change anything on GitHub without asking me first.
```

## GitHub

`hark github` uses `GITHUB_TOKEN`, or the token from `gh auth token`. Set `HARK_GITHUB_API` for GitHub Enterprise.

GitHub has no per-user webhooks, so hark polls every minute:

- `/notifications` for activity by others on issues and PRs you authored or watch. Threads authored by someone else are marked `watching`.
- `/users/{you}/events` for your own actions, which GitHub never notifies you about.

hark reads only unread notifications and marks each thread as read once it has printed it, so a restarted hark, or a second consumer, never sees the same notification twice. On startup it picks up unread notifications from the last hour, which covers the gap while a monitor is re-armed; older unread ones are left untouched. Threads hark does not report (releases, CI runs) stay unread. Opening a thread on github.com marks it as read too, so hark skips it.

Known limits:

- GitHub delays the events API by 30 seconds up to a few hours, so your own actions can show up late.
- Each poll reports only the latest comment on a thread. Two comments within a minute show as one.

| Kind | When |
|------|------|
| `issue.commented`, `pr.commented` | someone comments |
| `issue.closed`, `pr.closed`, `pr.merged` | someone, or you, closes or merges |
| `pr.opened` | you open a PR |
| `issue.updated`, `pr.updated` | other thread activity, with the notification reason as text |
| `error` | a request failed; hark keeps polling |

## Adding a provider

Implement `hark.Provider` and register a constructor in `providers` in `cmd/hark/main.go`:

```go
type Provider interface {
	Watch(ctx context.Context, emit func(Event)) error
}
```
