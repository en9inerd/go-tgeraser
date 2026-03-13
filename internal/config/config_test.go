package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTimePeriod(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"3*days", 3 * 86400, false},
		{"5*hours", 5 * 3600, false},
		{"10*minutes", 10 * 60, false},
		{"30*seconds", 30, false},
		{"2*weeks", 2 * 604800, false},
		{"1*months", 0, true},
		{"abc*days", 0, true},
		{"noduration", 0, true},
		{"", 0, true},
		{"3*", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTimePeriod(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTimePeriod(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseTimePeriod(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateMediaTypes(t *testing.T) {
	tests := []struct {
		name    string
		types   []string
		wantErr bool
	}{
		{"valid single", []string{"photo"}, false},
		{"valid multiple", []string{"photo", "video", "audio"}, false},
		{"all valid types", []string{"photo", "video", "audio", "voice", "video_note", "gif", "document", "media"}, false},
		{"invalid type", []string{"photo", "sticker"}, true},
		{"empty slice", []string{}, false},
		{"completely invalid", []string{"invalid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMediaTypes(tt.types)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMediaTypes(%v) error = %v, wantErr %v", tt.types, err, tt.wantErr)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home dir: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tilde prefix", "~/foo/bar", filepath.Join(home, "foo/bar")},
		{"absolute path", "/usr/local/bin", "/usr/local/bin"},
		{"relative path", "foo/bar", "foo/bar"},
		{"tilde only slash", "~/", home},
		{"tilde no slash", "~foo", "~foo"},
		{"dot path cleaned", "./foo/../bar", "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandPath(tt.input)
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		cfg, err := ParseConfig([]string{"tgeraser"}, func(string) string { return "" })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.EntityType != "chat" {
			t.Errorf("default EntityType = %q, want %q", cfg.EntityType, "chat")
		}
		if cfg.APIID != 0 {
			t.Errorf("default APIID = %d, want 0", cfg.APIID)
		}
		if cfg.Verbose {
			t.Error("default Verbose should be false")
		}
		if cfg.WipeEverything {
			t.Error("default WipeEverything should be false")
		}
	})

	t.Run("flags", func(t *testing.T) {
		args := []string{
			"tgeraser",
			"--api-id", "12345",
			"--api-hash", "abcdef",
			"--entity-type", "user",
			"-p", "alice,bob",
			"-v",
			"-w",
			"-o", "7*days",
			"-m", "photo,video",
			"--delete-conversation",
		}
		cfg, err := ParseConfig(args, func(string) string { return "" })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.APIID != 12345 {
			t.Errorf("APIID = %d, want 12345", cfg.APIID)
		}
		if cfg.APIHash != "abcdef" {
			t.Errorf("APIHash = %q, want %q", cfg.APIHash, "abcdef")
		}
		if cfg.EntityType != "user" {
			t.Errorf("EntityType = %q, want %q", cfg.EntityType, "user")
		}
		if len(cfg.Peers) != 2 || cfg.Peers[0] != "alice" || cfg.Peers[1] != "bob" {
			t.Errorf("Peers = %v, want [alice bob]", cfg.Peers)
		}
		if !cfg.Verbose {
			t.Error("Verbose should be true")
		}
		if !cfg.WipeEverything {
			t.Error("WipeEverything should be true")
		}
		if cfg.OlderThan != 7*86400 {
			t.Errorf("OlderThan = %d, want %d", cfg.OlderThan, 7*86400)
		}
		if len(cfg.MediaTypes) != 2 || cfg.MediaTypes[0] != "photo" || cfg.MediaTypes[1] != "video" {
			t.Errorf("MediaTypes = %v, want [photo video]", cfg.MediaTypes)
		}
		if !cfg.DeleteConversation {
			t.Error("DeleteConversation should be true")
		}
	})

	t.Run("env vars", func(t *testing.T) {
		env := map[string]string{
			"TG_API_ID":     "99999",
			"TG_API_HASH":   "envhash",
			"TG_SESSION_DIR": "/tmp/sessions/",
		}
		cfg, err := ParseConfig([]string{"tgeraser"}, func(key string) string { return env[key] })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.APIID != 99999 {
			t.Errorf("APIID from env = %d, want 99999", cfg.APIID)
		}
		if cfg.APIHash != "envhash" {
			t.Errorf("APIHash from env = %q, want %q", cfg.APIHash, "envhash")
		}
		if cfg.SessionDir != "/tmp/sessions" {
			t.Errorf("SessionDir from env = %q, want %q", cfg.SessionDir, "/tmp/sessions")
		}
	})

	t.Run("invalid entity type", func(t *testing.T) {
		_, err := ParseConfig([]string{"tgeraser", "--entity-type", "group"}, func(string) string { return "" })
		if err == nil {
			t.Error("expected error for invalid entity type")
		}
	})

	t.Run("invalid older-than", func(t *testing.T) {
		_, err := ParseConfig([]string{"tgeraser", "-o", "badvalue"}, func(string) string { return "" })
		if err == nil {
			t.Error("expected error for invalid older-than")
		}
	})

	t.Run("invalid media type", func(t *testing.T) {
		_, err := ParseConfig([]string{"tgeraser", "-m", "sticker"}, func(string) string { return "" })
		if err == nil {
			t.Error("expected error for invalid media type")
		}
	})

	t.Run("version flag", func(t *testing.T) {
		cfg, err := ParseConfig([]string{"tgeraser", "--version"}, func(string) string { return "" })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.ShowVersion {
			t.Error("ShowVersion should be true")
		}
	})
}
