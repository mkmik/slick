package slack

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantCh    string
		wantTS    string
		wantErr   bool
	}{
		{
			name:   "valid nvidia slack URL",
			url:    "https://nvidia.slack.com/archives/C08HFRFLRC4/p1771497400064149",
			wantCh: "C08HFRFLRC4",
			wantTS: "1771497400.064149",
		},
		{
			name:   "valid URL with query params",
			url:    "https://nvidia.slack.com/archives/C08HFRFLRC4/p1771497400064149?thread_ts=1771497400.064149",
			wantCh: "C08HFRFLRC4",
			wantTS: "1771497400.064149",
		},
		{
			name:   "different workspace",
			url:    "https://myteam.slack.com/archives/C12345ABCD/p1600000000000100",
			wantCh: "C12345ABCD",
			wantTS: "1600000000.000100",
		},
		{
			name:    "missing p prefix",
			url:     "https://nvidia.slack.com/archives/C08HFRFLRC4/1771497400064149",
			wantErr: true,
		},
		{
			name:    "wrong path structure",
			url:     "https://nvidia.slack.com/messages/C08HFRFLRC4",
			wantErr: true,
		},
		{
			name:    "not a URL",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := ParseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got ref=%+v", ref)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ref.ChannelID != tt.wantCh {
				t.Errorf("ChannelID = %q, want %q", ref.ChannelID, tt.wantCh)
			}
			if ref.Timestamp != tt.wantTS {
				t.Errorf("Timestamp = %q, want %q", ref.Timestamp, tt.wantTS)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		wantCh  string
		wantTS  string
		wantErr bool
	}{
		{
			name:   "thread permalink replies in thread",
			target: "https://nvidia.slack.com/archives/C08HFRFLRC4/p1771497400064149",
			wantCh: "C08HFRFLRC4",
			wantTS: "1771497400.064149",
		},
		{
			name:   "reply permalink uses parent thread_ts",
			target: "https://nvidia.slack.com/archives/C08HFRFLRC4/p1771497999123456?thread_ts=1771497400.064149",
			wantCh: "C08HFRFLRC4",
			wantTS: "1771497400.064149",
		},
		{
			name:   "bare channel ID posts top-level",
			target: "C08HFRFLRC4",
			wantCh: "C08HFRFLRC4",
		},
		{
			name:   "channel name posts top-level",
			target: "#platform-eng",
			wantCh: "#platform-eng",
		},
		{
			name:    "empty target",
			target:  "",
			wantErr: true,
		},
		{
			name:    "malformed slack URL",
			target:  "https://nvidia.slack.com/messages/C08HFRFLRC4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTarget(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got target=%+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ChannelID != tt.wantCh {
				t.Errorf("ChannelID = %q, want %q", got.ChannelID, tt.wantCh)
			}
			if got.ThreadTS != tt.wantTS {
				t.Errorf("ThreadTS = %q, want %q", got.ThreadTS, tt.wantTS)
			}
		})
	}
}
