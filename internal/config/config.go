package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	APIID              int
	APIHash            string
	SessionDir         string
	SessionName        string
	EntityType         string
	Peers              []string
	Limit              int
	WipeEverything     bool
	OlderThan          int // seconds
	DeleteConversation bool
	MediaTypes         []string
	Verbose            bool
	ShowVersion        bool
}

type credentials struct {
	APIID   int    `json:"api_id"`
	APIHash string `json:"api_hash"`
}

func ParseConfig(args []string, getenv func(string) string) (*Config, error) {
	getEnv := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}

	getEnvInt := func(key string, fallback int) int {
		if v := getenv(key); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return fallback
	}

	fs := flag.NewFlagSet("tgeraser", flag.ContinueOnError)

	apiID := fs.Int("api-id", getEnvInt("TG_API_ID", 0), "Telegram API ID")
	apiHash := fs.String("api-hash", getEnv("TG_API_HASH", ""), "Telegram API Hash")
	sessionDir := fs.String("directory", getEnv("TG_SESSION_DIR", "~/.tgeraser/"), "Session storage directory")
	fs.StringVar(sessionDir, "d", getEnv("TG_SESSION_DIR", "~/.tgeraser/"), "Session storage directory (shorthand)")
	sessionName := fs.String("session", "", "Session name")
	entityType := fs.String("entity-type", "chat", "Entity type: any, chat, channel, user")
	peers := fs.String("peers", "", "Comma-separated peer IDs or usernames")
	fs.StringVar(peers, "p", "", "Comma-separated peer IDs or usernames (shorthand)")
	limit := fs.Int("limit", 0, "Number of recent chats to show")
	fs.IntVar(limit, "l", 0, "Number of recent chats to show (shorthand)")
	wipeEverything := fs.Bool("wipe-everything", false, "Delete messages from all entities of the specified type")
	fs.BoolVar(wipeEverything, "w", false, "Delete messages from all entities (shorthand)")
	olderThan := fs.String("older-than", "", `Delete messages older than duration (e.g., "3*days", "5*hours")`)
	fs.StringVar(olderThan, "o", "", `Delete messages older than duration (shorthand)`)
	mediaType := fs.String("media-type", "", "Comma-separated media types: photo, video, audio, voice, video_note, gif, document, media")
	fs.StringVar(mediaType, "m", "", "Media type filter (shorthand)")
	deleteConversation := fs.Bool("delete-conversation", false, "Delete entire conversation (user peers only)")
	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	fs.BoolVar(verbose, "v", false, "Enable verbose logging (shorthand)")
	showVersion := fs.Bool("version", false, "Show version")

	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}

	var olderThanSec int
	if *olderThan != "" {
		parsed, err := ParseTimePeriod(*olderThan)
		if err != nil {
			return nil, fmt.Errorf("invalid --older-than value: %w", err)
		}
		olderThanSec = parsed
	}

	var peerList []string
	if *peers != "" {
		for p := range strings.SplitSeq(*peers, ",") {
			if p = strings.TrimSpace(p); p != "" {
				peerList = append(peerList, p)
			}
		}
	}

	var mediaTypes []string
	if *mediaType != "" {
		for m := range strings.SplitSeq(*mediaType, ",") {
			if m = strings.TrimSpace(strings.ToLower(m)); m != "" {
				mediaTypes = append(mediaTypes, m)
			}
		}
		if err := validateMediaTypes(mediaTypes); err != nil {
			return nil, err
		}
	}

	validEntityTypes := map[string]bool{"any": true, "chat": true, "channel": true, "user": true}
	if !validEntityTypes[*entityType] {
		return nil, fmt.Errorf("invalid entity type %q: must be one of: any, chat, channel, user", *entityType)
	}

	return &Config{
		APIID:              *apiID,
		APIHash:            *apiHash,
		SessionDir:         expandPath(*sessionDir),
		SessionName:        *sessionName,
		EntityType:         *entityType,
		Peers:              peerList,
		Limit:              *limit,
		WipeEverything:     *wipeEverything,
		OlderThan:          olderThanSec,
		DeleteConversation: *deleteConversation,
		MediaTypes:         mediaTypes,
		Verbose:            *verbose,
		ShowVersion:        *showVersion,
	}, nil
}

// ResolveCredentials loads API credentials from flags, env, file, or interactive prompt.
func (c *Config) ResolveCredentials() error {
	if c.APIID != 0 && c.APIHash != "" {
		return nil
	}

	if err := os.MkdirAll(c.SessionDir, 0o700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	credsPath := filepath.Join(c.SessionDir, "credentials.json")
	if data, err := os.ReadFile(credsPath); err == nil {
		var creds credentials
		if err := json.Unmarshal(data, &creds); err == nil && creds.APIID != 0 && creds.APIHash != "" {
			c.APIID = creds.APIID
			c.APIHash = creds.APIHash
			return nil
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Telegram API credentials not found. Obtain them from https://my.telegram.org/auth?to=apps")

	fmt.Print("Enter your API ID: ")
	if !scanner.Scan() {
		return errors.New("failed to read API ID")
	}
	apiID, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		return fmt.Errorf("invalid API ID: %w", err)
	}

	fmt.Print("Enter your API hash: ")
	if !scanner.Scan() {
		return errors.New("failed to read API hash")
	}
	apiHash := strings.TrimSpace(scanner.Text())
	if apiHash == "" {
		return errors.New("API hash cannot be empty")
	}

	c.APIID = apiID
	c.APIHash = apiHash

	fmt.Print("Save credentials to file? [y/n]: ")
	if scanner.Scan() && strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		creds := credentials{APIID: apiID, APIHash: apiHash}
		data, _ := json.MarshalIndent(creds, "", "    ")
		if err := os.WriteFile(credsPath, data, 0o600); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		fmt.Printf("Credentials saved to %s\n", credsPath)
	}

	return nil
}

// ResolveSession determines the session file path from flag, existing sessions, or interactive prompt.
func (c *Config) ResolveSession() (string, error) {
	if err := os.MkdirAll(c.SessionDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	if c.SessionName != "" {
		return filepath.Join(c.SessionDir, c.SessionName+".session"), nil
	}

	sessions := listSessions(c.SessionDir)
	scanner := bufio.NewScanner(os.Stdin)

	if len(sessions) > 0 {
		fmt.Println("\nAvailable sessions:")
		for i, s := range sessions {
			fmt.Printf("  %d. %s\n", i+1, s)
		}

		fmt.Print("\nChoose session number: ")
		if !scanner.Scan() {
			return "", errors.New("failed to read session choice")
		}
		num, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
		if err != nil || num < 1 || num > len(sessions) {
			return "", fmt.Errorf("invalid session number: %s", scanner.Text())
		}
		c.SessionName = sessions[num-1]
		return filepath.Join(c.SessionDir, c.SessionName+".session"), nil
	}

	fmt.Print("Enter session name: ")
	if !scanner.Scan() {
		return "", errors.New("failed to read session name")
	}
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		return "", errors.New("session name cannot be empty")
	}
	c.SessionName = name
	return filepath.Join(c.SessionDir, name+".session"), nil
}

func listSessions(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var sessions []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".session") {
			sessions = append(sessions, strings.TrimSuffix(e.Name(), ".session"))
		}
	}
	return sessions
}

var validMediaTypes = map[string]bool{
	"photo": true, "video": true, "audio": true, "voice": true,
	"video_note": true, "gif": true, "document": true, "media": true,
}

func validateMediaTypes(types []string) error {
	for _, t := range types {
		if !validMediaTypes[t] {
			valid := make([]string, 0, len(validMediaTypes))
			for k := range validMediaTypes {
				valid = append(valid, k)
			}
			return fmt.Errorf("invalid media type %q: valid types are %s", t, strings.Join(valid, ", "))
		}
	}
	return nil
}

func ParseTimePeriod(s string) (int, error) {
	parts := strings.SplitN(s, "*", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("expected format: NUMBER*UNIT (e.g., 3*days)")
	}

	value, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", parts[0], err)
	}

	multipliers := map[string]int{
		"seconds": 1,
		"minutes": 60,
		"hours":   3600,
		"days":    86400,
		"weeks":   604800,
	}
	mult, ok := multipliers[parts[1]]
	if !ok {
		return 0, fmt.Errorf("invalid time unit %q: use seconds, minutes, hours, days, or weeks", parts[1])
	}

	return value * mult, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return filepath.Clean(path)
}
