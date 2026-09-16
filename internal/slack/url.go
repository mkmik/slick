package slack

import (
	"cmp"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ThreadRef identifies a Slack thread by channel ID and message timestamp.
type ThreadRef struct {
	ChannelID string
	Timestamp string // e.g. "1771497400.064149"
}

var pathPattern = regexp.MustCompile(`^/archives/([A-Z0-9]+)/p(\d{10})(\d{6})$`)

// ParseURL extracts a ThreadRef from a Slack message permalink.
// Example: https://nvidia.slack.com/archives/C08HFRFLRC4/p1771497400064149
func ParseURL(rawURL string) (ThreadRef, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ThreadRef{}, fmt.Errorf("invalid URL: %w", err)
	}
	m := pathPattern.FindStringSubmatch(u.Path)
	if m == nil {
		return ThreadRef{}, fmt.Errorf("URL does not match Slack thread format: %s", rawURL)
	}
	return ThreadRef{
		ChannelID: m[1],
		Timestamp: m[2] + "." + m[3],
	}, nil
}

// Target identifies where a message should be posted. An empty ThreadTS means a
// new top-level message.
type Target struct {
	ChannelID string
	ThreadTS  string
}

// ParseTarget resolves a post target from a thread permalink, a bare channel ID,
// or a #channel-name.
func ParseTarget(s string) (Target, error) {
	if s == "" {
		return Target{}, fmt.Errorf("empty target")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		// ponytail: channel names go to Slack verbatim, which resolves them for
		// chat.postMessage. Add a conversations.list lookup if that proves flaky.
		return Target{ChannelID: s}, nil
	}
	ref, err := ParseURL(s)
	if err != nil {
		return Target{}, err
	}
	// A permalink to a reply carries its parent's timestamp in ?thread_ts=, which
	// ParseURL ignores. Prefer it: posting against a reply's own timestamp makes
	// Slack reparent the message and warn.
	threadTS := ref.Timestamp
	if u, err := url.Parse(s); err == nil {
		threadTS = cmp.Or(u.Query().Get("thread_ts"), threadTS)
	}
	return Target{ChannelID: ref.ChannelID, ThreadTS: threadTS}, nil
}
