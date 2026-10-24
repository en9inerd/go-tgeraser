package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/en9inerd/go-tgeraser/internal/config"
	"github.com/en9inerd/go-tgeraser/internal/eraser"
	"github.com/en9inerd/go-tgeraser/internal/log"
	"github.com/en9inerd/go-tgeraser/internal/login"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
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

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	waiter := floodwait.NewSimpleWaiter().WithMaxRetries(10).WithMaxWait(1 * time.Minute)

	var resolver dcs.Resolver
	if cfg.ProxyAddr != "" {
		resolver, err = dcs.MTProxy(cfg.ProxyAddr, cfg.ProxySecret, dcs.MTProxyOptions{})
		if err != nil {
			return fmt.Errorf("failed to build proxy resolver: %w", err)
		}
	}

	opts := telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionPath},
		Middlewares: []telegram.Middleware{
			waiter,
		},
		Device: telegram.DeviceConfig{
			DeviceModel:   runtime.GOOS + "/" + runtime.GOARCH,
			SystemVersion: runtime.Version(),
			AppVersion:    version,
		},
	}
	if resolver != nil {
		opts.Resolver = resolver
	}

	client := telegram.NewClient(cfg.APIID, cfg.APIHash, opts)

	fmt.Println("Connecting to Telegram servers...")
	return client.Run(ctx, func(ctx context.Context) error {
		if err := login.Authorize(ctx, client.Auth()); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		e := eraser.New(client.API(), cfg, logger)
		return e.Run(ctx)
	})
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
