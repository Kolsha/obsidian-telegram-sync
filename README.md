# Obsidian Telegram Sync

Standalone daemon that syncs Telegram messages to your Obsidian vault as Markdown files. No Obsidian required at runtime.

## Features

- Receives messages via Telegram Bot API (long polling)
- Converts Telegram formatting (bold, italic, code, links, etc.) to Markdown
- Supports all media types: photos, videos, voice messages, documents, audio, video notes
- Flexible template system for note paths and content (`{{content}}`, `{{user}}`, `{{chat}}`, `{{messageDate:FORMAT}}`, etc.)
- Message distribution rules with filters (by user, chat, topic, content, forward source)
- Append to existing notes or create new ones
- Media file download and storage
- Runs as a background daemon (systemd-compatible)
- Configuration via YAML file or environment variables

## Quick Start

### 1. Create a Telegram Bot

Talk to [@BotFather](https://t.me/botfather) to create a bot and get your token.

### 2. Configure

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your bot token and vault path
```

### 3. Run

```bash
# Build and run
make run

# Or with Go directly
go run ./cmd/obsidian-telegram-sync/ -config config.yaml
```

### 4. Docker

```bash
docker build -t obsidian-telegram-sync .
docker run --rm \
  -v /path/to/config.yaml:/app/config.yaml \
  -v /path/to/vault:/app/vault \
  obsidian-telegram-sync
```

## Configuration

See [config.example.yaml](config.example.yaml) for a full example.

| Setting | Description | Default |
|---------|-------------|---------|
| `bot_token` | Telegram bot token (required) | - |
| `allowed_chats` | List of allowed usernames or chat IDs | `[]` (all allowed) |
| `delete_messages` | Delete messages from Telegram after processing | `false` |
| `vault_path` | Path to your notes directory | `.` |
| `distribution_rules` | Message routing rules (see below) | one catch-all rule |
| `log_level` | Log level: debug, info, warn, error | `info` |

Environment variables override config file values with `OTS_` prefix (e.g., `OTS_BOT_TOKEN`).

## Distribution Rules

Rules are evaluated in order; first match wins.

```yaml
distribution_rules:
  - filter: "{{chat=Work}}"
    note_path: "Work/{{content:30}}.md"
    file_path: "Work/files/{{file:name}}.{{file:extension}}"
  - filter: "{{all}}"
    note_path: "Inbox/{{content:30}} - {{messageTime:YYYYMMDDHHmmss}}.md"
    file_path: "Inbox/files/{{file:name}}.{{file:extension}}"
```

`file_link_template` controls how a saved file is referenced in the note. Default: `![{{file:name}}.{{file:extension}}]({{file:path}})` (embedded — Obsidian renders images inline and audio/voice as a player). Use `[...]` instead of `![...]` for a plain link:

```yaml
  - filter: "{{all}}"
    file_path: "Inbox/files/{{file:name}}.{{file:extension}}"
    file_link_template: "[{{file:name}}.{{file:extension}}]({{file:path}})"
```

### Filter Syntax

| Filter | Description |
|--------|-------------|
| `{{all}}` | Match all messages |
| `{{user=name}}` | User equals name/ID |
| `{{user!=name}}` | User not equals |
| `{{chat=name}}` | Chat name/ID equals |
| `{{chat~text}}` | Chat name/ID contains |
| `{{content~text}}` | Message content contains |
| `{{forwardFrom=name}}` | Forwarded from user/chat |

Operations: `=` (equals), `!=` (not equals), `~` (contains), `!~` (not contains).

### Template Variables

| Variable | Description |
|----------|-------------|
| `{{content}}` | Full message text (Markdown) |
| `{{content:N}}` | First N characters |
| `{{content:[N-M]}}` | Lines N through M |
| `{{messageDate:FORMAT}}` | Message date (moment.js format) |
| `{{messageTime:FORMAT}}` | Same as messageDate |
| `{{date:FORMAT}}` | Current date |
| `{{user}}` | Link to sender |
| `{{user:name}}` | Sender username |
| `{{user:fullName}}` | Sender full name |
| `{{chat:name}}` | Chat name |
| `{{forwardFrom}}` | Forward source link |
| `{{files}}` | Embedded file links |
| `{{hashtag:[N]}}` | Nth hashtag from message |
| `{{file:type}}` | File type (photo, video, etc.) |
| `{{file:name}}` | Original file name (falls back to file type) |
| `{{file:extension}}` | File extension |
| `{{file:uniqueId}}` | Telegram file_unique_id |
| `{{file:path}}` | Vault-relative path of the saved file (file_link_template only) |

## Systemd Service

```ini
[Unit]
Description=Obsidian Telegram Sync
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/obsidian-telegram-sync -config /etc/obsidian-telegram-sync/config.yaml
Restart=on-failure
RestartSec=10
User=obsidian-sync

[Install]
WantedBy=multi-user.target
```

## Development

```bash
make test       # Run tests with race detector
make lint       # Run linter
make build      # Build binary
```

## License

GNU Affero General Public License v3.0
