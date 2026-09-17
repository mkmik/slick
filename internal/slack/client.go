package slack

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	goslack "github.com/slack-go/slack"
)

// Message is a simplified representation of a Slack message.
type Message struct {
	UserID   string
	Username string
	Text     string
	TS       string
	IsBot    bool
}

// Thread holds the messages in a Slack thread.
type Thread struct {
	Ref      ThreadRef
	Messages []Message
	Users    map[string]string // userID -> display name, for resolving inline mentions
}

// slackAPI abstracts the Slack API methods we use, enabling test mocks.
type slackAPI interface {
	AuthTest() (*goslack.AuthTestResponse, error)
	GetConversationReplies(params *goslack.GetConversationRepliesParameters) ([]goslack.Message, bool, string, error)
	GetUserInfo(userID string) (*goslack.User, error)
	PostMessage(channelID string, options ...goslack.MsgOption) (string, string, error)
	GetPermalink(params *goslack.PermalinkParameters) (string, error)
}

// AuthTest calls the Slack auth.test API to verify the token is valid.
func (c *Client) AuthTest() (*goslack.AuthTestResponse, error) {
	return c.api.AuthTest()
}

// Client wraps the Slack API with user caching.
type Client struct {
	api       slackAPI
	transport *transport
	userCache map[string]string
}

// IsBotToken reports whether a token posts as an app rather than as a person.
// Slack's token prefixes encode exactly that: xoxb- is a bot token, while xoxp-
// and xoxc- act as the authenticating user.
//
// Anything unrecognised is treated as a user token, so a message that might be
// attributed to a human is never silently stripped of its tooling footer.
//
// ponytail: prefix check, swap in auth.test's BotID if a token ever lies.
func IsBotToken(token string) bool {
	return strings.HasPrefix(token, "xoxb-")
}

// New creates a Client with the given Slack API token.
func New(token string) *Client {
	t := &transport{}
	return &Client{
		api:       goslack.New(token, goslack.OptionHTTPClient(&http.Client{Transport: t})),
		transport: t,
		userCache: make(map[string]string),
	}
}

// Scopes returns the OAuth scopes Slack reported on the most recent API call,
// empty if none have been seen. Slack only reports them in a response header.
func (c *Client) Scopes() string {
	c.transport.mu.Lock()
	defer c.transport.mu.Unlock()
	return c.transport.scopes
}

// transport records the OAuth scopes Slack reports back on each response.
type transport struct {
	mu     sync.Mutex
	scopes string
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if resp != nil {
		if s := resp.Header.Get("x-oauth-scopes"); s != "" {
			t.mu.Lock()
			t.scopes = s
			t.mu.Unlock()
		}
	}
	return resp, err
}

// FetchThread fetches all messages in a Slack thread identified by the given URL.
func (c *Client) FetchThread(rawURL string) (*Thread, error) {
	ref, err := ParseURL(rawURL)
	if err != nil {
		return nil, err
	}

	var allMsgs []goslack.Message
	cursor := ""
	for {
		params := &goslack.GetConversationRepliesParameters{
			ChannelID: ref.ChannelID,
			Timestamp: ref.Timestamp,
			Cursor:    cursor,
			Limit:     200,
			Inclusive:  true,
		}
		msgs, hasMore, nextCursor, err := c.api.GetConversationReplies(params)
		if err != nil {
			return nil, fmt.Errorf("fetching thread replies: %w", err)
		}
		allMsgs = append(allMsgs, msgs...)
		if !hasMore {
			break
		}
		cursor = nextCursor
	}

	// Collect all unique user IDs from authors and inline mentions.
	userIDs := make(map[string]struct{})
	mentionRe := regexp.MustCompile(`<@([A-Z0-9]+)(?:\|[^>]*)?>`)
	for _, m := range allMsgs {
		if m.User != "" {
			userIDs[m.User] = struct{}{}
		}
		for _, match := range mentionRe.FindAllStringSubmatch(m.Text, -1) {
			userIDs[match[1]] = struct{}{}
		}
	}

	// Resolve all users upfront.
	for uid := range userIDs {
		c.resolveUser(uid)
	}

	thread := &Thread{
		Ref:   ref,
		Users: make(map[string]string, len(c.userCache)),
	}
	for k, v := range c.userCache {
		thread.Users[k] = v
	}

	for _, m := range allMsgs {
		thread.Messages = append(thread.Messages, c.convertMessage(m))
	}
	return thread, nil
}

// Post sends text to t and returns a permalink to the new message. The text is
// expected to already be Slack mrkdwn; see markdown.ToMrkdwn.
func (c *Client) Post(t Target, text string) (string, error) {
	opts := []goslack.MsgOption{goslack.MsgOptionText(text, false)}
	if t.ThreadTS != "" {
		opts = append(opts, goslack.MsgOptionTS(t.ThreadTS))
	}
	channelID, ts, err := c.api.PostMessage(t.ChannelID, opts...)
	if err != nil {
		return "", fmt.Errorf("posting message: %w", err)
	}
	// The message is already out; a failed permalink lookup must not make the
	// command look like it failed. Fall back to the raw identifiers.
	link, err := c.api.GetPermalink(&goslack.PermalinkParameters{Channel: channelID, Ts: ts})
	if err != nil {
		return channelID + "/" + ts, nil
	}
	return link, nil
}

func (c *Client) resolveUser(userID string) string {
	if name, ok := c.userCache[userID]; ok {
		return name
	}
	info, err := c.api.GetUserInfo(userID)
	if err != nil {
		c.userCache[userID] = userID
		return userID
	}
	name := info.Profile.DisplayName
	if name == "" {
		name = info.Profile.RealName
	}
	if name == "" {
		name = userID
	}
	c.userCache[userID] = name
	return name
}

func (c *Client) convertMessage(m goslack.Message) Message {
	msg := Message{
		UserID: m.User,
		Text:   m.Text,
		TS:     m.Timestamp,
		IsBot:  m.BotID != "",
	}
	if msg.IsBot {
		msg.Username = m.Username
		if msg.Username == "" {
			msg.Username = "bot"
		}
	} else if m.User != "" {
		msg.Username = c.resolveUser(m.User)
	} else {
		msg.Username = "unknown"
	}
	return msg
}
