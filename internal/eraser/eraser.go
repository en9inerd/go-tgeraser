package eraser

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/en9inerd/go-tgeraser/internal/config"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

type Eraser struct {
	api      *tg.Client
	sender   *message.Sender
	self     *tg.User
	cfg      *config.Config
	logger   *slog.Logger
	entities []entity
}

func New(api *tg.Client, self *tg.User, cfg *config.Config, logger *slog.Logger) *Eraser {
	return &Eraser{
		api:    api,
		sender: message.NewSender(api),
		self:   self,
		cfg:    cfg,
		logger: logger,
	}
}

func (e *Eraser) Run(ctx context.Context) error {
	if err := e.determineEntities(ctx); err != nil {
		return fmt.Errorf("failed to determine entities: %w", err)
	}

	if len(e.entities) == 0 {
		fmt.Println("No entities to process.")
		return nil
	}

	startTime := time.Now()
	fmt.Printf("\nDeletion started at: %s (local)\n", startTime.Format(time.RFC3339))

	if err := e.deleteMessagesFromEntities(ctx); err != nil {
		return err
	}

	finishTime := time.Now()
	fmt.Printf("Deletion finished at: %s (local)\n", finishTime.Format(time.RFC3339))
	fmt.Printf("Duration: %s\n\n", finishTime.Sub(startTime).Round(time.Second))

	e.entities = nil
	return nil
}

func (e *Eraser) determineEntities(ctx context.Context) error {
	if len(e.cfg.Peers) > 0 {
		return e.getEntitiesByPeers(ctx)
	}
	if e.cfg.WipeEverything {
		return e.loadFilteredEntities(ctx)
	}
	return e.getUserSelectedEntity(ctx)
}

func (e *Eraser) getEntitiesByPeers(ctx context.Context) error {
	for _, peer := range e.cfg.Peers {
		ent, err := e.resolvePeer(ctx, peer)
		if err != nil {
			return fmt.Errorf("failed to resolve peer %q: %w", peer, err)
		}
		e.entities = append(e.entities, *ent)
	}
	return nil
}

func (e *Eraser) resolvePeer(ctx context.Context, peer string) (*entity, error) {
	if !isNumeric(peer) {
		username := strings.TrimPrefix(peer, "@")
		resolved, err := e.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: username,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to resolve username %q: %w", username, err)
		}
		return entityFromResolved(resolved)
	}

	// For numeric IDs, fetch all dialogs (ignore --limit) to find the entity
	peerID, _ := strconv.ParseInt(peer, 10, 64)
	dialogs, err := e.getAllDialogs(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}
	for _, d := range dialogs {
		if d.id == peerID {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("entity with ID %d not found in dialogs", peerID)
}

func (e *Eraser) loadFilteredEntities(ctx context.Context) error {
	dialogs, err := e.getAllDialogs(ctx, e.cfg.Limit)
	if err != nil {
		return err
	}
	e.entities = filterByType(dialogs, e.cfg.EntityType)
	return nil
}

func (e *Eraser) getUserSelectedEntity(ctx context.Context) error {
	dialogs, err := e.getAllDialogs(ctx, e.cfg.Limit)
	if err != nil {
		return err
	}

	filtered := filterByType(dialogs, e.cfg.EntityType)
	if len(filtered) == 0 {
		return fmt.Errorf("no entities of type %q found", e.cfg.EntityType)
	}

	printHeader("List of entities")
	for i, ent := range filtered {
		fmt.Printf("  %d. %s\t | %d\n", i+1, ent.displayName, ent.id)
	}

	fmt.Print("\nChoose peer: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}
	num, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || num < 1 || num > len(filtered) {
		return fmt.Errorf("invalid choice: %s", scanner.Text())
	}

	chosen := filtered[num-1]
	fmt.Printf("Chosen: %s\n", chosen.displayName)
	e.entities = []entity{chosen}
	return nil
}
