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

func TestLossy(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "plain text is faithful",
			input: "on it",
		},
		{
			name:  "bold, links and headings are faithful",
			input: "# Report\n**done** see [logs](https://ci/1)",
		},
		{
			name:  "table",
			input: "| a | b |\n|---|---|\n| 1 | 2 |",
			want:  []string{"tables"},
		},
		{
			name:  "padded table with alignment row",
			input: "| a   | b   |\n|:--- | ---:|\n| 1   | 2   |",
			want:  []string{"tables"},
		},
		{
			name:  "image",
			input: "![diagram](https://example.com/d.png)",
			want:  []string{"images"},
		},
		{
			name:  "task list",
			input: "- [ ] first\n- [x] second",
			want:  []string{"task lists"},
		},
		{
			name:  "nested list",
			input: "- outer\n  - inner",
			want:  []string{"nested lists"},
		},
		{
			name:  "flat list is faithful",
			input: "- one\n- two",
		},
		{
			name:  "a table inside a code fence is intentional, not lossy",
			input: "```\n| a | b |\n|---|---|\n```",
		},
		{
			name:  "several at once",
			input: "| a | b |\n|---|---|\n\n![x](https://e.com/x.png)",
			want:  []string{"tables", "images"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lossy(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("Lossy(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Lossy(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestToMrkdwn(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "bold",
			input: "**deploy done**",
			want:  "*deploy done*",
		},
		{
			name:  "underscore bold",
			input: "__deploy done__",
			want:  "*deploy done*",
		},
		{
			name:  "link inside a sentence",
			input: "see [logs](https://ci/1) for details",
			want:  "see <https://ci/1|logs> for details",
		},
		{
			name:  "strikethrough",
			input: "~~old plan~~ new plan",
			want:  "~old plan~ new plan",
		},
		{
			name:  "heading",
			input: "## Deploy report\nall green",
			want:  "*Deploy report*\nall green",
		},
		{
			name:  "escapes ampersand and angle brackets",
			input: "tom & jerry, 3 < 5, 7 > 2",
			want:  "tom &amp; jerry, 3 &lt; 5, 7 &gt; 2",
		},
		{
			name:  "blockquote survives escaping",
			input: "> quoted line\nplain line",
			want:  "> quoted line\nplain line",
		},
		{
			name:  "inline code is not reformatted",
			input: "run `make **all**` now",
			want:  "run `make **all**` now",
		},
		{
			name:  "fenced block is not reformatted but is escaped",
			input: "before\n```\nif a < b && c {\n  **x**\n}\n```\nafter **bold**",
			want:  "before\n```\nif a &lt; b &amp;&amp; c {\n  **x**\n}\n```\nafter *bold*",
		},
		{
			name:  "italics and list markers are left alone",
			input: "- _first_ item\n- second",
			want:  "- _first_ item\n- second",
		},
		{
			name:  "ampersand in a link URL is escaped",
			input: "[build](https://ci/job?a=1&b=2)",
			want:  "<https://ci/job?a=1&amp;b=2|build>",
		},
		{
			name:  "plain text is unchanged",
			input: "on it",
			want:  "on it",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToMrkdwn(tt.input); got != tt.want {
				t.Errorf("ToMrkdwn(%q)\n got = %q\nwant = %q", tt.input, got, tt.want)
			}
		})
	}
}
