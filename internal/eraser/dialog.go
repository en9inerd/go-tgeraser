package eraser

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"
)

// getAllDialogs fetches dialogs from the Telegram API using gotd's query iterator.
// Pass limit=0 to fetch all dialogs.
func (e *Eraser) getAllDialogs(ctx context.Context, limit int) ([]entity, error) {
	var allEntities []entity

	err := query.GetDialogs(e.api).BatchSize(100).ForEach(ctx, func(ctx context.Context, elem dialogs.Elem) error {
		if limit > 0 && len(allEntities) >= limit {
			return errLimitReached
		}

		ent := entityFromDialogElem(elem)
		if ent != nil {
			allEntities = append(allEntities, *ent)
		}
		return nil
	})

	if err != nil && err != errLimitReached {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	if limit > 0 && len(allEntities) > limit {
		allEntities = allEntities[:limit]
	}

	return allEntities, nil
}

// sentinel error to stop ForEach iteration when limit is reached
var errLimitReached = fmt.Errorf("limit reached")

func entityFromDialogElem(elem dialogs.Elem) *entity {
	switch p := elem.Dialog.GetPeer().(type) {
	case *tg.PeerUser:
		if user, ok := elem.Entities.User(p.UserID); ok {
			return &entity{
				peer:        elem.Peer,
				displayName: userDisplayName(user),
				peerType:    "user",
				isSelf:      user.Self,
				id:          user.ID,
			}
		}
	case *tg.PeerChat:
		if chat, ok := elem.Entities.Chat(p.ChatID); ok {
			return &entity{
				peer:        elem.Peer,
				displayName: chat.Title,
				peerType:    "chat",
				id:          chat.ID,
			}
		}
	case *tg.PeerChannel:
		if ch, ok := elem.Entities.Channel(p.ChannelID); ok {
			return &entity{
				peer:        elem.Peer,
				displayName: ch.Title,
				peerType:    "channel",
				isMegagroup: ch.Megagroup || ch.Gigagroup,
				id:          ch.ID,
			}
		}
	}
	return nil
}
