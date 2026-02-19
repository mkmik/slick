package markdown

import (
	"strings"
	"testing"

	slackclient "mkm.pub/slick/internal/slack"
)

func TestConvertMrkdwn(t *testing.T) {
	users := map[string]string{
		"U123ABC": "alice",
		"U456DEF": "bob",
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "user mention with display name",
			input: "hey <@U123ABC|alice> check this",
			want:  "hey @alice check this",
		},
		{
			name:  "user mention without display name, resolved from map",
			input: "hey <@U123ABC> check this",
			want:  "hey @alice check this",
		},
		{
			name:  "user mention unknown user",
			input: "hey <@U999ZZZ> check this",
			want:  "hey @U999ZZZ check this",
		},
		{
			name:  "channel mention with name",
			input: "see <#C123ABC|general>",
			want:  "see #general",
		},
		{
			name:  "channel mention without name",
			input: "see <#C123ABC>",
			want:  "see #C123ABC",
		},
		{
			name:  "link with label",
			input: "check <https://example.com|this link>",
			want:  "check [this link](https://example.com)",
		},
		{
			name:  "link without label",
			input: "check <https://example.com/path>",
			want:  "check https://example.com/path",
		},
		{
			name:  "multiple tokens",
			input: "<@U123ABC> shared <https://example.com|a link> in <#C123ABC|general>",
			want:  "@alice shared [a link](https://example.com) in #general",
		},
		{
			name:  "plain text unchanged",
			input: "just plain text",
			want:  "just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertMrkdwn(tt.input, users)
			if got != tt.want {
				t.Errorf("ConvertMrkdwn(%q) =\n  %q\nwant:\n  %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	thread := &slackclient.Thread{
		Messages: []slackclient.Message{
			{Username: "alice", Text: "The deployment is broken"},
			{Username: "bob", Text: "Can you share the logs?"},
			{Username: "alice", Text: "Here they are"},
		},
		Users: map[string]string{},
	}

	got := Render(thread)

	if !strings.Contains(got, "## Thread") {
		t.Error("expected '## Thread' header")
	}
	if !strings.Contains(got, "## Replies") {
		t.Error("expected '## Replies' header")
	}
	if !strings.Contains(got, "**alice**: The deployment is broken") {
		t.Error("expected thread root message")
	}
	if !strings.Contains(got, "**bob**: Can you share the logs?") {
		t.Error("expected first reply")
	}
	if !strings.Contains(got, "---") {
		t.Error("expected separator between thread and replies")
	}
}

func TestRenderSingleMessage(t *testing.T) {
	thread := &slackclient.Thread{
		Messages: []slackclient.Message{
			{Username: "alice", Text: "Just a standalone message"},
		},
		Users: map[string]string{},
	}

	got := Render(thread)

	if !strings.Contains(got, "## Thread") {
		t.Error("expected '## Thread' header")
	}
	if strings.Contains(got, "## Replies") {
		t.Error("should not have '## Replies' for single message")
	}
}
