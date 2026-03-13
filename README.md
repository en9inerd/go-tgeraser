# go-tgeraser

Go implementation of [TgEraser](https://github.com/en9inerd/tgeraser) — a tool that deletes all your messages from a chat, channel, or conversation on Telegram without requiring admin privileges.

## Installation

Download a pre-built binary from the [Releases](https://github.com/en9inerd/go-tgeraser/releases) page, or build from source:

```
go install github.com/en9inerd/go-tgeraser/cmd/tgeraser@latest
```

## Configuration

You'll need `api_id` and `api_hash`, which you can obtain from [my.telegram.org](https://my.telegram.org/auth?to=apps).

There are three ways to provide credentials:
1. **CLI flags**: `--api-id` and `--api-hash`
2. **Environment variables**: `TG_API_ID` and `TG_API_HASH`
3. **Credentials file**: The tool will prompt you on first run and optionally save to `~/.tgeraser/credentials.json`

Credentials file format:
```json
{
    "api_id": 111111,
    "api_hash": "abcdef1234567890abcdef1234567890"
}
```

## Usage

```
tgeraser [flags]

Flags:
    --api-id INT                Telegram API ID (or TG_API_ID env var)
    --api-hash STRING           Telegram API Hash (or TG_API_HASH env var)
    -d, --directory PATH        Session storage directory (default: ~/.tgeraser/)
    --session NAME              Session name
    --entity-type TYPE          Entity type: any, chat, channel, user (default: chat)
    -p, --peers PEER_ID         Comma-separated peer IDs or usernames
    -l, --limit NUM             Number of recent chats to show
    -w, --wipe-everything       Delete messages from all entities of the specified type
    --delete-conversation       Delete entire conversation (user peers only)
    -o, --older-than STRING     Delete messages older than duration (e.g., "3*days", "5*hours")
    -m, --media-type TYPES      Comma-separated media types: photo, video, audio, voice,
                                video_note, gif, document, media
    -v, --verbose               Enable verbose logging
    --version                   Show version
```

Running the tool without `--peers` or `--wipe-everything` will show an interactive list of your chats to choose from.

### Examples

Delete all your messages from a specific chat by username:
```
tgeraser --session myaccount -p @chatname
```

Delete messages older than 7 days from all chats:
```
tgeraser --session myaccount -w -o "7*days"
```

Delete only photos and videos from a specific peer:
```
tgeraser --session myaccount -p @chatname -m "photo,video"
```

Delete entire conversation with a user:
```
tgeraser --session myaccount -p @username --entity-type user --delete-conversation
```

## Contributing

If you have any issues or suggestions, please feel free to open an issue or submit a pull request.
