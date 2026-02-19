---
name: slick
description: >-
  Fetch Slack thread content using the slick CLI. Triggers on Slack thread URLs
  matching https://<workspace>.slack.com/archives/<channel_id>/p<timestamp>,
  or when the user mentions "slick", "slack thread", or "fetch slack".
version: 0.1.0
user-invocable: true
---

# slick - Fetch Slack Threads as Markdown

Use the `slick` CLI to fetch Slack thread conversations and include them as context in the current conversation.

## When to Trigger

Activate this skill when the user pastes or references a Slack thread URL.

Slack thread URLs follow this format:

```
https://<workspace>.slack.com/archives/<channel_id>/p<timestamp>
```

Where:
- `<workspace>` is the Slack workspace subdomain (any valid subdomain)
- `<channel_id>` is an alphanumeric channel or DM identifier (e.g. `C08HFRFLRC4`, `D087M2B6MA6`)
- `<timestamp>` is a 16-digit message timestamp composed of a 10-digit Unix epoch followed by 6 digits of microseconds (e.g. `p1771497400064149`)

The URL may also include query parameters (e.g. `?thread_ts=...`), which should be preserved when passing to `slick`.

## Installation

Check if `slick` is already installed:

```bash
command -v slick
```

If `slick` is not found, install it from GitHub releases:

```bash
OS=$(uname -s)
ARCH=$(uname -m)

case "${ARCH}" in
  aarch64) ARCH="arm64" ;;
esac

EXT="tar.gz"

ARCHIVE="slick_${OS}_${ARCH}.${EXT}"
URL="https://github.com/mkmik/slick/releases/latest/download/${ARCHIVE}"

INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "${INSTALL_DIR}"

TMP=$(mktemp -d)
curl -fSL "${URL}" -o "${TMP}/${ARCHIVE}"
tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"
install -m 755 "${TMP}/slick" "${INSTALL_DIR}/slick"
rm -rf "${TMP}"
```

If `~/.local/bin` is not on `PATH`, add it:

```bash
export PATH="${HOME}/.local/bin:${PATH}"
```

Verify installation:

```bash
slick --version
```

If the binary download fails (e.g. unsupported platform), fall back to installing from source:

```bash
go install mkm.pub/slick@latest
```

## Prerequisites

The `slick` CLI requires a Slack API token. Before running any slick command, verify the `SLICK_TOKEN` environment variable is set:

```bash
echo "${SLICK_TOKEN:?SLICK_TOKEN is not set}"
```

If `SLICK_TOKEN` is not set, inform the user:

> The `SLICK_TOKEN` environment variable is required but not set.
> Please set it to a valid Slack API token with the necessary scopes and export it in your shell profile:
>
> ```bash
> export SLICK_TOKEN=xoxb-your-token-here
> ```

**Do not** proceed with `slick cat` if the token is missing.

## Usage

Fetch the thread and include the output as conversation context:

```bash
slick cat "<slack_thread_url>"
```

Always quote the URL to handle query parameters correctly.

The output is markdown with the thread structure (original message under `## Thread`, replies under `## Replies`) and resolved `@user` mentions.

## Error Handling

- **Missing token**: `SLICK_TOKEN` is not set or empty. Ask the user to set it.
- **Invalid URL**: The URL doesn't match the Slack thread permalink format. Ask the user for the correct URL.
- **API/network errors**: Slack API may be unreachable or the token may lack required scopes. Suggest the user verify their token and network connectivity.
- **Installation failure**: If both binary download and `go install` fail, direct the user to https://github.com/mkmik/slick/releases to download manually.
