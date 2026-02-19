# slick
CLI to fetch context from slack threads

## Install

```bash
go install mkm.pub/slick@latest
```

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
