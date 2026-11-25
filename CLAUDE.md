# hark

hark prints one line per event from GitHub, Jira, Slack and Todoist, so an agent can run it as a background monitor. Go, for Linux and macOS.

## Documenting a change

hark is documented in three places. A change people can notice updates all three in the same PR as the code:

- `cmd/hark/help.txt`: the reference `hark help` prints, written for agents.
- `README.md`: the reference for people.
- `docs/index.html`: the landing page, served by GitHub Pages from `main`, `/docs`.

People notice a new or changed provider, event kind, field of the line, flag, command, environment variable, `hark doctor` check, state file, or how a restart resumes.

### A new provider

- `help.txt`: the opening sentence, the `<provider> is ...` list in USAGE, FOR AGENTS (how it resumes, what never to do on it), PROVIDER, REF and @ACTOR in OUTPUT, an example line, its entry in KINDS, "Your own actions", the default start under `--since`, its `--filter` syntax with examples, ENVIRONMENT and STATE.
- `README.md`: the opening sentence, the `hark doctor` sample, the prompt in "Use it from Claude Code" (which monitor to start and what to do with each kind), the `--since` paragraph, the `--filter` table, and a section of its own with setup steps, how it polls, a kinds table and its known limits.
- `docs/index.html`: `<meta name="description">` and `og:description`, the hero title and lead, a `hark <provider>` tag and an example line in the hero stream, the field descriptions under "One event, one line", the environment variables and the `hark doctor` sample in setup step 2, the prompt in step 3, a card under "What each provider reports" (kinds, how it resumes, a `--filter` example), and "Restarts lose nothing".

### A new kind, flag, command or variable

Update what each of the three files already says about that provider or option. A new kind also goes in the prompt, in README and in the landing, with what the agent should do when it sees one.

### Copies that must stay equal

Setup step 3 on the landing holds the README prompt, HTML-escaped. After changing the prompt, this has to print `prompts match`:

~~~sh
python3 -c 'import html,re;r=re.search(r"^```text\n(.*?)\n```$",open("README.md").read(),re.S|re.M).group(1);d=html.unescape(re.search(r"id=\"prompt\"[^>]*>(.*?)</pre>",open("docs/index.html").read(),re.S).group(1));print("prompts match" if r==d else "prompts differ")'
~~~

The `hark doctor` samples in README and in setup step 2 show the same checks, in the order doctor prints them: providers sorted by name.

Examples use made-up data (`dukex/foo`, `acme/lib`, `PROJ-7`, `example.atlassian.net`, `example.slack.com`), never real repos, people or tokens.

### The landing page

Its tokens live in `:root`: dark theme only, yellow only on actions and selected states, Inter. Look at it at 390px wide and with JavaScript off, where every setup step shows and the copy buttons hide.

## Commits and releases

Conventional commits. Merging to `main` releases by commit type: `feat:` cuts a minor version and `fix:` a patch, while `docs:`, `ci:`, `chore:` and `test:` release nothing. A PR that only changes documentation uses `docs:`.
