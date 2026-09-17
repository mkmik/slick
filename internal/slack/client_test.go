package slack

import "testing"

func TestIsBotToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "bot token", token: "xoxb-123-456-abcdef", want: true},
		{name: "user token", token: "xoxp-123-456-789-abcdef", want: false},
		{name: "session token", token: "xoxc-123-456-abcdef", want: false},
		{name: "app-level token", token: "xapp-1-A123-456-abcdef", want: false},
		// Unrecognised tokens must read as user tokens: guessing "bot" would drop
		// the footer from a message attributed to a person.
		{name: "unknown prefix", token: "nonsense", want: false},
		{name: "empty", token: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBotToken(tt.token); got != tt.want {
				t.Errorf("IsBotToken(%q) = %v, want %v", tt.token, got, tt.want)
			}
		})
	}
}
