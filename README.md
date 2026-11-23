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
go install github.com/dukex/hark/cmd/hark@latest
```

## GitHub

`hark github` uses `GITHUB_TOKEN`, or the token from `gh auth token`. Set `HARK_GITHUB_API` for GitHub Enterprise.

GitHub has no per-user webhooks, so hark polls every minute:

- `/notifications` for activity by others on issues and PRs you authored or watch. Threads authored by someone else are marked `watching`.
- `/users/{you}/events` for your own actions, which GitHub never notifies you about.

Only events that happen after hark starts are printed. Nothing is marked as read.

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
