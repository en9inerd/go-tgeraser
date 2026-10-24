package config

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/en9inerd/go-pkgs/flagpair"
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
	ProxyAddr          string
	ProxySecret        []byte
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

	r := flagpair.New("tgeraser")

	sessionDir := r.String("directory", "d", getEnv("TG_SESSION_DIR", "~/.tgeraser/"), "Session storage directory")
	sessionName := r.String("session", "", "", "Session name")
	entityType := r.String("entity-type", "", "chat", "Entity type: any, chat, channel, user")
	peers := r.String("peers", "p", "", "Comma-separated peer IDs or usernames")
	limit := r.Int("limit", "l", 0, "Number of recent chats to show")
	wipeEverything := r.Bool("wipe-everything", "w", false, "Delete messages from all entities of the specified type")
	olderThan := r.String("older-than", "o", "", `Delete messages older than duration (e.g., "3*days", "5*hours")`)
	mediaType := r.String("media-type", "m", "", "Comma-separated media types: photo, video, audio, voice, video_note, gif, document, media")
	deleteConversation := r.Bool("delete-conversation", "", false, "Delete entire conversation (user peers only)")
	proxy := r.String("proxy", "", "", "MTProto proxy (HOST:PORT:SECRET)")
	verbose := r.Bool("verbose", "v", false, "Enable verbose logging")
	showVersion := r.Bool("version", "", false, "Show version")

	if err := r.Parse(args[1:]); err != nil {
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

	if *deleteConversation && len(mediaTypes) > 0 {
		return nil, errors.New("--delete-conversation cannot be combined with --media-type")
	}

	validEntityTypes := map[string]bool{"any": true, "chat": true, "channel": true, "user": true}
	if !validEntityTypes[*entityType] {
		return nil, fmt.Errorf("invalid entity type %q: must be one of: any, chat, channel, user", *entityType)
	}

	var proxyAddr string
	var proxySecret []byte
	if *proxy != "" {
		addr, secret, err := parseProxy(*proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid --proxy value: %w", err)
		}
		proxyAddr, proxySecret = addr, secret
	}

	return &Config{
		APIID:              getEnvInt("TG_API_ID", 0),
		APIHash:            getEnv("TG_API_HASH", ""),
		SessionDir:         expandPath(*sessionDir),
		SessionName:        *sessionName,
		EntityType:         *entityType,
		Peers:              peerList,
		Limit:              *limit,
		WipeEverything:     *wipeEverything,
		OlderThan:          olderThanSec,
		DeleteConversation: *deleteConversation,
		MediaTypes:         mediaTypes,
		ProxyAddr:          proxyAddr,
		ProxySecret:        proxySecret,
		Verbose:            *verbose,
		ShowVersion:        *showVersion,
	}, nil
}

// ResolveCredentials loads API credentials from env, file, or interactive prompt.
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

	fmt.Print("Enter session name (any label for the local session file): ")
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
			valid := slices.Sorted(maps.Keys(validMediaTypes))
			return fmt.Errorf("invalid media type %q: valid types are %s", t, strings.Join(valid, ", "))
		}
	}
	return nil
}

func parseProxy(s string) (addr string, secret []byte, err error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return "", nil, fmt.Errorf("must be HOST:PORT:SECRET")
	}
	host, portStr, secretHex := parts[0], parts[1], parts[2]
	if host == "" {
		return "", nil, fmt.Errorf("host cannot be empty")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return "", nil, fmt.Errorf("invalid port %q: must be 1-65535", portStr)
	}
	secret, err = hex.DecodeString(secretHex)
	if err != nil {
		return "", nil, fmt.Errorf("invalid secret: %w", err)
	}
	return fmt.Sprintf("%s:%d", host, port), secret, nil
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
	if value <= 0 {
		return 0, fmt.Errorf("duration must be positive, got %q", parts[0])
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

	const maxPeriodSec = 20 * 365 * 86400
	if value > maxPeriodSec/mult {
		return 0, fmt.Errorf("duration %q is too large: maximum is 20 years", s)
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
