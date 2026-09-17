# slick
CLI to fetch context from slack threads

## Install

```bash
go install mkm.pub/slick/cmd/slick@latest
```

## Usage

Set `SLICK_TOKEN` to a Slack token. Reading needs `channels:history` (plus
`groups:history` for private channels) and `users:read`; posting also needs
`chat:write`.

Read a thread as markdown:

```bash
slick cat https://acme.slack.com/archives/C08HFRFLRC4/p1771497400064149
```

Post a message. The target is a thread permalink (replies in that thread), a
channel ID, a `#channel-name`, or `@me` for a DM to yourself; the body comes
from `-m` or stdin:

```bash
slick post '#platform-eng' -m 'deploy done' -y
slick post https://acme.slack.com/archives/C08HFRFLRC4/p1771497400064149 -m 'on it' -y
slick post @me -m 'remember to redeploy' -y
cat report.md | slick post C08HFRFLRC4 -y
```

`post` never sends silently. Without `-y` it prints a preview of exactly what
would go out, then asks `send? [y/N]` if you are on a terminal. When there is
nobody to ask — a script, a pipeline, an agent — it exits non-zero instead, so
`-y` stays the only way through unattended. Markdown in
the body is converted to Slack's mrkdwn (`**bold**` → `*bold*`,
`[t](u)` → `<u|t>`), and a successful send prints the new message's permalink —
which you can hand straight back to `slick cat`.

By default the body is converted to Slack's mrkdwn, which has no table syntax.
Pass `--markdown` to send it as standard Markdown in a Block Kit markdown block
and let Slack render it — tables, headings and nested lists all work. It renders
subtly differently from a normal message, so it is opt-in rather than default.

Messages sent with a user token get a `_Sent using_ :magic:` footer, so readers
can tell a message attributed to you was sent by a tool rather than typed in
Slack. Bot tokens (`xoxb-`) skip it — those messages are already visibly from an
app. Pass `--no-disclaimer` to omit it anyway.

# demo

The second half of this README.md was produced with:

```bash
slick cat https://acme.slack.com/archives/D087M2B6MA6/p1771515440176589 >>README.md
```

---

## Thread

**John Doe**: I'd like to have a simple CLI tool that fetches a thread from slack and renders it as markdown

---

## Replies

**Claudia Dakota**: It's easy today; just tell claude code to create it with this prompt:

```
create a go CLI utility whose main goal is to fetch context from slack threads and format it so that it LLM agents can make sense of it (markdown)

1. go module `mkm.pub/slick`
2. use kong for cli along with https://github.com/jotaen/kong-completion
3. use go 1.26
4. create github workflow to build and test. On merge tag with semantic version and  release binaries of the tool with goreleaser
5. Use slack token from SLICK_TOKEN env var (use kong to map flags to env vars)
6. Implement at least one command `slick cat <url>` if the url is https://foo.slack.com/archives/C08HFRFLRC4/p1771497400064149 it would use that thread ID and that timestamp id, fetch all the messages, resolve all the user handles and reformat as markdown
```

**John Doe**: Wow, that's slick!
