package telegram

import (
	"path"
	"strings"

	"github.com/go-telegram/bot/models"
)

var fileTypes = []string{"photo", "video", "voice", "document", "audio", "video_note"}

// FileInfo holds information about a file attached to a message.
type FileInfo struct {
	Type      string
	FileID    string
	FileSize  int64
	FileName  string
	MimeType  string
	UniqueID  string
}

// GetFileInfo extracts file information from a message.
// Returns nil if the message has no file attachment.
func GetFileInfo(msg *models.Message) *FileInfo {
	if msg.Photo != nil && len(msg.Photo) > 0 {
		largest := msg.Photo[len(msg.Photo)-1]
		return &FileInfo{
			Type:     "photo",
			FileID:   largest.FileID,
			FileSize: int64(largest.FileSize),
			UniqueID: largest.FileUniqueID,
		}
	}
	if msg.Video != nil {
		return &FileInfo{
			Type:     "video",
			FileID:   msg.Video.FileID,
			FileSize: msg.Video.FileSize,
			FileName: msg.Video.FileName,
			MimeType: msg.Video.MimeType,
			UniqueID: msg.Video.FileUniqueID,
		}
	}
	if msg.Voice != nil {
		return &FileInfo{
			Type:     "voice",
			FileID:   msg.Voice.FileID,
			FileSize: msg.Voice.FileSize,
			MimeType: msg.Voice.MimeType,
			UniqueID: msg.Voice.FileUniqueID,
		}
	}
	if msg.Document != nil {
		return &FileInfo{
			Type:     "document",
			FileID:   msg.Document.FileID,
			FileSize: msg.Document.FileSize,
			FileName: msg.Document.FileName,
			MimeType: msg.Document.MimeType,
			UniqueID: msg.Document.FileUniqueID,
		}
	}
	if msg.Audio != nil {
		return &FileInfo{
			Type:     "audio",
			FileID:   msg.Audio.FileID,
			FileSize: msg.Audio.FileSize,
			FileName: msg.Audio.FileName,
			MimeType: msg.Audio.MimeType,
			UniqueID: msg.Audio.FileUniqueID,
		}
	}
	if msg.VideoNote != nil {
		return &FileInfo{
			Type:     "video_note",
			FileID:   msg.VideoNote.FileID,
			FileSize: int64(msg.VideoNote.FileSize),
			UniqueID: msg.VideoNote.FileUniqueID,
		}
	}
	return nil
}

// FileExtension resolves the file extension in priority order: original file
// name, Telegram remote file path (from GetFile), MIME type, per-type default.
func (fi *FileInfo) FileExtension(remotePath string) string {
	if fi.FileName != "" {
		if ext := strings.TrimPrefix(path.Ext(fi.FileName), "."); ext != "" {
			return ext
		}
	}
	if remotePath != "" {
		if ext := strings.TrimPrefix(path.Ext(remotePath), "."); ext != "" {
			return ext
		}
	}
	if ext, ok := mimeExtensions[fi.MimeType]; ok {
		return ext
	}
	if ext, ok := typeDefaultExtensions[fi.Type]; ok {
		return ext
	}
	return "file"
}

// BaseName returns the file name without extension.
func (fi *FileInfo) BaseName() string {
	if fi.FileName != "" {
		ext := path.Ext(fi.FileName)
		return strings.TrimSuffix(fi.FileName, ext)
	}
	return fi.Type
}

var mimeExtensions = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/gif":       "gif",
	"image/webp":      "webp",
	"video/mp4":       "mp4",
	"video/mpeg":      "mpeg",
	"audio/mpeg":      "mp3",
	"audio/ogg":       "ogg",
	"audio/mp4":       "m4a",
	"application/pdf": "pdf",
	"application/zip": "zip",
}

// Bot API omits mime_type for these types; extensions are fixed by Telegram.
var typeDefaultExtensions = map[string]string{
	"photo":      "jpg",
	"voice":      "ogg",
	"video_note": "mp4",
}

// IsAllowedChat checks if the message sender is in the allowed chats list.
func IsAllowedChat(msg *models.Message, allowedChats []string) bool {
	if len(allowedChats) == 0 {
		return true
	}
	for _, allowed := range allowedChats {
		if allowed == "" {
			continue
		}
		if msg.From != nil && msg.From.Username == allowed {
			return true
		}
		chatID := strings.TrimLeft(strings.Replace(
			strings.Replace(string(rune(msg.Chat.ID)), "-", "", 1), " ", "", -1,
		), "0")
		_ = chatID

		chatIDStr := formatInt64(msg.Chat.ID)
		if chatIDStr == allowed {
			return true
		}
	}
	return false
}

func formatInt64(n int64) string {
	if n < 0 {
		return "-" + formatUint64(uint64(-n))
	}
	return formatUint64(uint64(n))
}

func formatUint64(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
