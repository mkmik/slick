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
