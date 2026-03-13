package eraser

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/tg"
)

type entity struct {
	peer        tg.InputPeerClass
	displayName string
	peerType    string // raw Telegram type: "user", "chat", "channel"
	isSelf      bool
	isMegagroup bool
	id          int64
}

func entityFromResolved(resolved *tg.ContactsResolvedPeer) (*entity, error) {
	entities := peer.EntitiesFromResult(resolved)
	inputPeer, err := entities.ExtractPeer(resolved.Peer)
	if err != nil {
		return nil, fmt.Errorf("could not extract peer: %w", err)
	}

	switch p := resolved.Peer.(type) {
	case *tg.PeerUser:
		if user, ok := entities.User(p.UserID); ok {
			return &entity{
				peer:        inputPeer,
				displayName: userDisplayName(user),
				peerType:    "user",
				isSelf:      user.Self,
				id:          user.ID,
			}, nil
		}
	case *tg.PeerChat:
		if chat, ok := entities.Chat(p.ChatID); ok {
			return &entity{
				peer:        inputPeer,
				displayName: chat.Title,
				peerType:    "chat",
				id:          chat.ID,
			}, nil
		}
	case *tg.PeerChannel:
		if ch, ok := entities.Channel(p.ChannelID); ok {
			return &entity{
				peer:        inputPeer,
				displayName: ch.Title,
				peerType:    "channel",
				isMegagroup: ch.Megagroup || ch.Gigagroup,
				id:          ch.ID,
			}, nil
		}
	}
	return nil, fmt.Errorf("could not extract entity from resolved peer")
}

func filterByType(entities []entity, entityType string) []entity {
	var filtered []entity
	for _, ent := range entities {
		switch entityType {
		case "any":
			filtered = append(filtered, ent)
		case "user":
			if ent.peerType == "user" && !ent.isSelf {
				filtered = append(filtered, ent)
			}
		case "chat":
			if ent.peerType == "chat" || (ent.peerType == "channel" && ent.isMegagroup) {
				filtered = append(filtered, ent)
			}
		case "channel":
			if ent.peerType == "channel" && !ent.isMegagroup {
				filtered = append(filtered, ent)
			}
		}
	}
	return filtered
}

func userDisplayName(u *tg.User) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	if name == "" {
		return u.Username
	}
	return name
}

func isNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

func printHeader(title string) {
	border := strings.Repeat("=", len(title))
	fmt.Printf("\n==%s==\n", border)
	fmt.Printf("= %s =\n", title)
	fmt.Printf("==%s==\n", border)
}

func mediaFilter(mediaType string) tg.MessagesFilterClass {
	switch mediaType {
	case "photo":
		return &tg.InputMessagesFilterPhotos{}
	case "video":
		return &tg.InputMessagesFilterVideo{}
	case "audio":
		return &tg.InputMessagesFilterMusic{}
	case "voice":
		return &tg.InputMessagesFilterVoice{}
	case "video_note":
		return &tg.InputMessagesFilterRoundVideo{}
	case "gif":
		return &tg.InputMessagesFilterGif{}
	case "document":
		return &tg.InputMessagesFilterDocument{}
	default:
		return nil
	}
}

func expandMediaTypes(types []string) []string {
	allTypes := []string{"photo", "video", "audio", "voice", "video_note", "gif", "document"}
	if slices.Contains(types, "media") {
		return allTypes
	}
	return types
}
