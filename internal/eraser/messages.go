package eraser

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"
)

const deleteBatchSize = 100

func (e *Eraser) deleteMessagesFromEntities(ctx context.Context) error {
	var maxDate int
	if e.cfg.OlderThan > 0 {
		maxDate = int(time.Now().UTC().Add(-time.Duration(e.cfg.OlderThan) * time.Second).Unix())
	}

	for _, ent := range e.entities {
		if ent.peerType == "user" && e.cfg.DeleteConversation {
			if err := e.deleteConversation(ctx, ent); err != nil {
				return err
			}
			continue
		}

		printHeader(fmt.Sprintf("Getting messages from '%s'...", ent.displayName))
		msgIDs, err := e.getMessagesToDelete(ctx, ent, maxDate)
		if err != nil {
			return fmt.Errorf("failed to get messages from %q: %w", ent.displayName, err)
		}

		fmt.Printf("\nFound %d messages to delete.\n", len(msgIDs))
		if len(msgIDs) > 0 {
			if err := e.deleteMessages(ctx, ent, msgIDs); err != nil {
				return fmt.Errorf("failed to delete messages from %q: %w", ent.displayName, err)
			}
		}
	}
	return nil
}

func (e *Eraser) deleteConversation(ctx context.Context, ent entity) error {
	printHeader(fmt.Sprintf("Deleting entire conversation with user '%s'...", ent.displayName))

	for {
		req := &tg.MessagesDeleteHistoryRequest{
			Peer:  ent.peer,
			MaxID: 0,
		}
		req.SetRevoke(true)

		result, err := e.api.MessagesDeleteHistory(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to delete conversation with %q: %w", ent.displayName, err)
		}
		if result.Offset <= 0 {
			break
		}
	}

	fmt.Printf("\nDeleted entire conversation with user '%s'.\n\n", ent.displayName)
	return nil
}

func (e *Eraser) getMessagesToDelete(ctx context.Context, ent entity, maxDate int) ([]int, error) {
	if len(e.cfg.MediaTypes) == 0 {
		return e.searchMessages(ctx, ent.peer, &tg.InputMessagesFilterEmpty{}, maxDate)
	}

	types := expandMediaTypes(e.cfg.MediaTypes)
	idSet := make(map[int]struct{})
	for _, mediaType := range types {
		filter := mediaFilter(mediaType)
		if filter == nil {
			continue
		}
		fmt.Printf("  Fetching %s...\n", mediaType)
		ids, err := e.searchMessages(ctx, ent.peer, filter, maxDate)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			idSet[id] = struct{}{}
		}
	}

	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids, nil
}

func (e *Eraser) searchMessages(ctx context.Context, peer tg.InputPeerClass, filter tg.MessagesFilterClass, maxDate int) ([]int, error) {
	var allIDs []int

	builder := query.Messages(e.api).Search(peer).
		FromID(&tg.InputPeerSelf{}).
		Filter(filter).
		BatchSize(deleteBatchSize)

	if maxDate > 0 {
		builder = builder.MaxDate(maxDate)
	}

	err := builder.ForEach(ctx, func(_ context.Context, elem messages.Elem) error {
		allIDs = append(allIDs, elem.Msg.GetID())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("message search failed: %w", err)
	}

	e.logger.Debug("fetched messages", "total", len(allIDs))
	return allIDs, nil
}

func (e *Eraser) deleteMessages(ctx context.Context, ent entity, msgIDs []int) error {
	printHeader(fmt.Sprintf("Deleting messages from '%s'...", ent.displayName))

	totalDeleted := 0
	for i := 0; i < len(msgIDs); i += deleteBatchSize {
		end := min(i+deleteBatchSize, len(msgIDs))
		batch := msgIDs[i:end]

		deleted, err := e.deleteBatch(ctx, ent, batch)
		if err != nil {
			return err
		}
		totalDeleted += deleted
	}

	fmt.Printf("\nDeleted %d messages of %d in '%s' entity.\n", totalDeleted, len(msgIDs), ent.displayName)
	if totalDeleted < len(msgIDs) {
		fmt.Printf("Remaining %d messages can't be deleted without admin rights because they are service messages.\n",
			len(msgIDs)-totalDeleted)
	}
	fmt.Println()
	return nil
}

func (e *Eraser) deleteBatch(ctx context.Context, ent entity, ids []int) (int, error) {
	result, err := e.sender.To(ent.peer).Revoke().Messages(ctx, ids...)
	if err != nil {
		return 0, fmt.Errorf("message deletion failed: %w", err)
	}
	return result.PtsCount, nil
}
