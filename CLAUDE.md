# hark

hark prints one line per event from Datadog, GitHub, Jira, Slack and Todoist, so an agent can run it as a background monitor. Go, for Linux and macOS.

## Documenting a change

hark is documented in three places. A change people can notice updates all three in the same PR as the code:

- `cmd/hark/help.txt`: the reference `hark help` prints, written for agents.
- `README.md`: the reference for people.
- `docs/index.html`: the site at https://hark.dukex.sh, served by GitHub Pages from `main`, `/docs`. `docs/CNAME` holds the domain.

People notice a new or changed provider, event kind, field of the line, flag, command, environment variable, `hark doctor` check, state file, or how a restart resumes.

### A new provider

- `help.txt`: the opening sentence, the `<provider> is ...` list in USAGE, FOR AGENTS (how it resumes, what never to do on it), PROVIDER, REF and @ACTOR in OUTPUT, an example line, its entry in KINDS, "Your own actions", the default start under `--since`, its `--filter` syntax with examples, ENVIRONMENT and STATE.
- `README.md`: the opening sentence, the `hark doctor` sample, the prompt in "Use it from Claude Code" (which monitor to start and what to do with each kind), the `--since` paragraph, the `--filter` table, and a section of its own with setup steps, how it polls, a kinds table and its known limits.
- `docs/index.html`: the list of services in `<meta name="description">`, `og:description`, `twitter:description` and the JSON-LD `description`, in the hero lead and in the "Your agent hears from" strip; a card under "What your agent does with each event" when the service brings a new kind of work; its variables in the "Connect your accounts" step; the `hark doctor` sample in the "Your agent starts listening" step; the prompt in the "Paste the prompt into Claude Code" step; the PROVIDER field under "One line per event, written for agents"; and a tab with its panel under "One monitor per service" (kinds, what is good to know, a `--filter` example).
- The GitHub repo description, which also names every provider: `gh repo edit dukex/hark --description "..."`, run by Emerson after the merge.

### A new kind, flag, command or variable

Update what each of the three files already says about that provider or option. A new kind also goes in the prompt, in README and in the landing, with what the agent should do when it sees one.

### Copies that must stay equal

The "Paste the prompt into Claude Code" step on the site holds the README prompt, HTML-escaped, and every "Copy the setup prompt" button copies it from there. After changing the prompt, this has to print `prompts match`:

~~~sh
python3 -c 'import html,re;r=re.search(r"^```text\n(.*?)\n```$",open("README.md").read(),re.S|re.M).group(1);d=html.unescape(re.search(r"id=\"prompt\"[^>]*>(.*?)</pre>",open("docs/index.html").read(),re.S).group(1));print("prompts match" if r==d else "prompts differ")'
~~~

The `hark doctor` samples in README and in the "Your agent starts listening" step show the same checks, in the order doctor prints them: providers sorted by name.

Examples use made-up data (`dukex/foo`, `acme/lib`, `acme/checkout`, `PROJ-7`, `example.atlassian.net`, `example.slack.com`), never real repos, people or tokens.

### The site

It sells what hark does for the agent; the line format is how it works, not the pitch. Its tokens live in `:root`: dark theme only, yellow only on actions, selected states and the logo, Inter, and the 56px and 40px display sizes (the hero title and the closing call to action) are the only type outside the design system's scale. Look at it at 390, 768, 1024 and 1440px wide and with JavaScript off, where every setup step and every service panel shows and the copy buttons hide.

`og.png` (1200×630, the link preview) and `apple-touch-icon.png` don't name the services, so a new provider leaves them alone. `sitemap.xml` and `robots.txt` point at https://hark.dukex.sh.

## Commits and releases

Conventional commits. Merging to `main` releases by commit type: `feat:` cuts a minor version and `fix:` a patch, while `docs:`, `ci:`, `chore:` and `test:` release nothing. A PR that only changes documentation uses `docs:`.
