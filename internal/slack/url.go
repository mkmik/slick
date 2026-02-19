package slack

import (
	"fmt"
	"net/url"
	"regexp"
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
