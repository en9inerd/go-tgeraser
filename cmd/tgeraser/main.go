package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/en9inerd/go-tgeraser/internal/config"
	"github.com/en9inerd/go-tgeraser/internal/eraser"
	"github.com/en9inerd/go-tgeraser/internal/log"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

var version = "dev"

func versionString() string {
	var revision, buildTime string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				if len(kv.Value) >= 7 {
					revision = kv.Value[:7]
				}
			case "vcs.time":
				buildTime = kv.Value
			}
		}
	}
	s := "tgeraser version " + version
	if revision != "" {
		s += " (" + revision + ")"
	}
	if buildTime != "" {
		s += " built " + buildTime
	}
	return s
}

func run(ctx context.Context, args []string, getenv func(string) string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.ParseConfig(args, getenv)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.ShowVersion {
		fmt.Println(versionString())
		return nil
	}

	logger := log.NewLogger(cfg.Verbose)
	logger.Info("starting go-tgeraser", "version", version)

	if err := cfg.ResolveCredentials(); err != nil {
		return fmt.Errorf("failed to resolve credentials: %w", err)
	}

	sessionPath, err := cfg.ResolveSession()
	if err != nil {
		return fmt.Errorf("failed to resolve session: %w", err)
	}

	waiter := floodwait.NewSimpleWaiter().WithMaxRetries(10)

	client := telegram.NewClient(cfg.APIID, cfg.APIHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionPath},
		Middlewares: []telegram.Middleware{
			waiter,
		},
		Device: telegram.DeviceConfig{
			DeviceModel:   runtime.GOOS + "/" + runtime.GOARCH,
			SystemVersion: runtime.Version(),
			AppVersion:    version,
		},
	})

	fmt.Println("Connecting to Telegram servers...")
	return client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(
			&terminalAuth{},
			auth.SendCodeOptions{},
		)
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		self, err := client.Self(ctx)
		if err != nil {
			return fmt.Errorf("failed to get self: %w", err)
		}

		e := eraser.New(client.API(), self, cfg, logger)
		return e.Run(ctx)
	})
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// terminalAuth implements auth.UserAuthenticator for interactive terminal sessions.
type terminalAuth struct{}

func (a *terminalAuth) Phone(_ context.Context) (string, error) {
	return prompt("Enter your phone number: ")
}

func (a *terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	return prompt("Enter the code you just received: ")
}

func (a *terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Two-step verification is enabled. Enter your password: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(password), nil
}

func (a *terminalAuth) AcceptTermsOfService(_ context.Context, _ tg.HelpTermsOfService) error {
	return nil
}

func (a *terminalAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign-up is not supported; use an existing Telegram account")
}

func prompt(message string) (string, error) {
	fmt.Print(message)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		return "", fmt.Errorf("no input received")
	}
	return strings.TrimSpace(scanner.Text()), nil
}
