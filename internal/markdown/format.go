package markdown

import (
	"fmt"
	"regexp"
	"strings"

	slackclient "mkm.pub/slick/internal/slack"
)

var (
	mentionRe = regexp.MustCompile(`<@([A-Z0-9]+)(?:\|([^>]*))?>`)
	channelRe = regexp.MustCompile(`<#([A-Z0-9]+)(?:\|([^>]*))?>`)
	linkRe    = regexp.MustCompile(`<(https?://[^|>]+)(?:\|([^>]*))?>`)
)

var (
	codeRe      = regexp.MustCompile("(?s)```.*?```|`[^`]*`")
	mdLinkRe    = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]+)\)`)
	mdBoldRe    = regexp.MustCompile(`\*\*([^*]+)\*\*|__([^_]+)__`)
	mdStrikeRe  = regexp.MustCompile(`~~([^~]+)~~`)
	mdHeadingRe = regexp.MustCompile(`(?m)^#{1,6}[ \t]+(.+?)[ \t]*$`)
	mdQuoteRe   = regexp.MustCompile(`(?m)^([ \t]*)&gt;`)
)

// Render converts a Thread into a markdown string suitable for LLM consumption.
func Render(thread *slackclient.Thread) string {
	var sb strings.Builder
	for i, msg := range thread.Messages {
		if i == 0 {
			sb.WriteString("## Thread\n\n")
		} else if i == 1 {
			sb.WriteString("---\n\n## Replies\n\n")
		}
		text := ConvertMrkdwn(msg.Text, thread.Users)
		fmt.Fprintf(&sb, "**%s**: %s\n\n", msg.Username, text)
	}
	return sb.String()
}

// ConvertMrkdwn converts Slack mrkdwn tokens to standard markdown.
func ConvertMrkdwn(s string, users map[string]string) string {
	// Resolve user mentions: <@U123|display> or <@U123>
	s = mentionRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := mentionRe.FindStringSubmatch(m)
		if parts[2] != "" {
			return "@" + parts[2]
		}
		if name, ok := users[parts[1]]; ok {
			return "@" + name
		}
		return "@" + parts[1]
	})

	// Resolve channel mentions: <#C123|channel-name>
	s = channelRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := channelRe.FindStringSubmatch(m)
		if parts[2] != "" {
			return "#" + parts[2]
		}
		return "#" + parts[1]
	})

	// Resolve links: <https://url|label> → [label](url), <https://url> → url
	s = linkRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := linkRe.FindStringSubmatch(m)
		if parts[2] != "" {
			return fmt.Sprintf("[%s](%s)", parts[2], parts[1])
		}
		return parts[1]
	})

	return s
}

// ToMrkdwn converts standard markdown to Slack mrkdwn, the inverse of
// ConvertMrkdwn. Code spans and fenced blocks are passed through untouched.
//
// Single-asterisk italics and list markers are left alone: *italic* would
// collide with the bold output, and Slack renders "- item" acceptably as-is.
func ToMrkdwn(s string) string {
	// Slack parses entities inside code spans too, so escape the whole string
	// first and convert only the formatting outside code. & must go first.
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")

	var sb strings.Builder
	last := 0
	for _, loc := range codeRe.FindAllStringIndex(s, -1) {
		sb.WriteString(toMrkdwn(s[last:loc[0]]))
		sb.WriteString(s[loc[0]:loc[1]])
		last = loc[1]
	}
	sb.WriteString(toMrkdwn(s[last:]))
	return sb.String()
}

// toMrkdwn converts a single already-escaped run of text known to contain no code.
func toMrkdwn(s string) string {
	s = mdQuoteRe.ReplaceAllString(s, "$1>") // blockquotes survive the escaping above
	s = mdLinkRe.ReplaceAllString(s, "<${2}|${1}>")
	s = mdBoldRe.ReplaceAllString(s, "*${1}${2}*")
	s = mdStrikeRe.ReplaceAllString(s, "~${1}~")
	s = mdHeadingRe.ReplaceAllString(s, "*${1}*")
	return s
}
