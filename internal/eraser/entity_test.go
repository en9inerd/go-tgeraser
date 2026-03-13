package eraser

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestFilterByType(t *testing.T) {
	entities := []entity{
		{displayName: "Alice", peerType: "user", isSelf: false},
		{displayName: "Saved Messages", peerType: "user", isSelf: true},
		{displayName: "Go Developers", peerType: "chat"},
		{displayName: "News Channel", peerType: "channel", isMegagroup: false},
		{displayName: "Big Group", peerType: "channel", isMegagroup: true},
	}

	tests := []struct {
		entityType string
		wantNames  []string
	}{
		{"any", []string{"Alice", "Saved Messages", "Go Developers", "News Channel", "Big Group"}},
		{"user", []string{"Alice"}},
		{"chat", []string{"Go Developers", "Big Group"}},
		{"channel", []string{"News Channel"}},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			got := filterByType(entities, tt.entityType)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("filterByType(%q) returned %d entities, want %d", tt.entityType, len(got), len(tt.wantNames))
			}
			for i, name := range tt.wantNames {
				if got[i].displayName != name {
					t.Errorf("filterByType(%q)[%d].displayName = %q, want %q", tt.entityType, i, got[i].displayName, name)
				}
			}
		})
	}
}

func TestExpandMediaTypes(t *testing.T) {
	t.Run("media expands to all", func(t *testing.T) {
		got := expandMediaTypes([]string{"media"})
		want := []string{"photo", "video", "audio", "voice", "video_note", "gif", "document"}
		if len(got) != len(want) {
			t.Fatalf("expandMediaTypes([media]) returned %d types, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("expandMediaTypes([media])[%d] = %q, want %q", i, got[i], want[i])
			}
		}
	})

	t.Run("media mixed with others still expands", func(t *testing.T) {
		got := expandMediaTypes([]string{"photo", "media"})
		if len(got) != 7 {
			t.Errorf("expandMediaTypes([photo, media]) returned %d types, want 7", len(got))
		}
	})

	t.Run("specific types pass through", func(t *testing.T) {
		input := []string{"photo", "video"}
		got := expandMediaTypes(input)
		if len(got) != 2 || got[0] != "photo" || got[1] != "video" {
			t.Errorf("expandMediaTypes(%v) = %v, want %v", input, got, input)
		}
	})
}

func TestUserDisplayName(t *testing.T) {
	tests := []struct {
		name  string
		user  *tg.User
		want  string
	}{
		{"first and last", &tg.User{FirstName: "John", LastName: "Doe"}, "John Doe"},
		{"first only", &tg.User{FirstName: "Alice"}, "Alice"},
		{"last only", &tg.User{LastName: "Smith"}, "Smith"},
		{"username fallback", &tg.User{Username: "jdoe"}, "jdoe"},
		{"first+last preferred over username", &tg.User{FirstName: "John", LastName: "Doe", Username: "jdoe"}, "John Doe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := userDisplayName(tt.user)
			if got != tt.want {
				t.Errorf("userDisplayName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"12345", true},
		{"-99", true},
		{"0", true},
		{"@username", false},
		{"abc", false},
		{"12.34", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isNumeric(tt.input); got != tt.want {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMediaFilter(t *testing.T) {
	tests := []struct {
		input   string
		wantNil bool
		wantType string
	}{
		{"photo", false, "*tg.InputMessagesFilterPhotos"},
		{"video", false, "*tg.InputMessagesFilterVideo"},
		{"audio", false, "*tg.InputMessagesFilterMusic"},
		{"voice", false, "*tg.InputMessagesFilterVoice"},
		{"video_note", false, "*tg.InputMessagesFilterRoundVideo"},
		{"gif", false, "*tg.InputMessagesFilterGif"},
		{"document", false, "*tg.InputMessagesFilterDocument"},
		{"unknown", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mediaFilter(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("mediaFilter(%q) should be nil", tt.input)
				}
				return
			}
			if got == nil {
				t.Fatalf("mediaFilter(%q) returned nil, want non-nil", tt.input)
			}
		})
	}
}
