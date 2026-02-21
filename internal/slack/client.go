package slack

import (
	"fmt"
	"regexp"

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
}

// AuthTest calls the Slack auth.test API to verify the token is valid.
func (c *Client) AuthTest() (*goslack.AuthTestResponse, error) {
	return c.api.AuthTest()
}

// Client wraps the Slack API with user caching.
type Client struct {
	api       slackAPI
	userCache map[string]string
}

// New creates a Client with the given Slack API token.
func New(token string) *Client {
	return &Client{
		api:       goslack.New(token),
		userCache: make(map[string]string),
	}
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
