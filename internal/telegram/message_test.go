package telegram

import (
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileInfo_Photo(t *testing.T) {
	msg := &models.Message{
		Photo: []models.PhotoSize{
			{FileID: "small", FileUniqueID: "s1", Width: 100, Height: 100, FileSize: 1000},
			{FileID: "large", FileUniqueID: "l1", Width: 800, Height: 800, FileSize: 50000},
		},
	}
	fi := GetFileInfo(msg)
	require.NotNil(t, fi)
	assert.Equal(t, "photo", fi.Type)
	assert.Equal(t, "large", fi.FileID)
}

func TestGetFileInfo_Document(t *testing.T) {
	msg := &models.Message{
		Document: &models.Document{
			FileID:       "doc1",
			FileUniqueID: "d1",
			FileName:     "report.pdf",
			MimeType:     "application/pdf",
			FileSize:     100000,
		},
	}
	fi := GetFileInfo(msg)
	require.NotNil(t, fi)
	assert.Equal(t, "document", fi.Type)
	assert.Equal(t, "report.pdf", fi.FileName)
	assert.Equal(t, "pdf", fi.FileExtension(""))
	assert.Equal(t, "report", fi.BaseName())
}

func TestGetFileInfo_Voice(t *testing.T) {
	msg := &models.Message{
		Voice: &models.Voice{
			FileID:       "v1",
			FileUniqueID: "vu1",
			MimeType:     "audio/ogg",
			FileSize:     5000,
		},
	}
	fi := GetFileInfo(msg)
	require.NotNil(t, fi)
	assert.Equal(t, "voice", fi.Type)
	assert.Equal(t, "ogg", fi.FileExtension(""))
}

func TestGetFileInfo_NoFile(t *testing.T) {
	msg := &models.Message{Text: "just text"}
	assert.Nil(t, GetFileInfo(msg))
}

func TestFileExtension_FromMime(t *testing.T) {
	fi := &FileInfo{Type: "photo", MimeType: "image/jpeg"}
	assert.Equal(t, "jpg", fi.FileExtension(""))
}

func TestFileExtension_UnknownMime(t *testing.T) {
	fi := &FileInfo{Type: "document", MimeType: "application/octet-stream"}
	assert.Equal(t, "file", fi.FileExtension(""))
}

func TestFileExtension_FromRemotePath(t *testing.T) {
	fi := &FileInfo{Type: "photo", UniqueID: "abc123"}
	assert.Equal(t, "jpg", fi.FileExtension("photos/file_9.jpg"))
}

func TestFileExtension_FileNameOverRemotePath(t *testing.T) {
	fi := &FileInfo{Type: "document", FileName: "notes.txt"}
	assert.Equal(t, "txt", fi.FileExtension("documents/file_1.bin"))
}

func TestBaseName_NoFileName(t *testing.T) {
	fi := &FileInfo{Type: "photo", UniqueID: "abc123"}
	assert.Equal(t, "photo", fi.BaseName())
}

func TestFileExtension_PhotoWithoutMime(t *testing.T) {
	fi := &FileInfo{Type: "photo", UniqueID: "abc123"}
	assert.Equal(t, "jpg", fi.FileExtension(""))
}

func TestFileExtension_VoiceWithoutMime(t *testing.T) {
	fi := &FileInfo{Type: "voice", UniqueID: "abc123"}
	assert.Equal(t, "ogg", fi.FileExtension(""))
}

func TestFileExtension_VideoNoteWithoutMime(t *testing.T) {
	fi := &FileInfo{Type: "video_note", UniqueID: "abc123"}
	assert.Equal(t, "mp4", fi.FileExtension(""))
}

func TestIsAllowedChat_Username(t *testing.T) {
	msg := &models.Message{
		From: &models.User{Username: "johndoe"},
		Chat: models.Chat{ID: 123},
	}
	assert.True(t, IsAllowedChat(msg, []string{"johndoe"}))
	assert.False(t, IsAllowedChat(msg, []string{"alice"}))
}

func TestIsAllowedChat_ChatID(t *testing.T) {
	msg := &models.Message{
		From: &models.User{Username: "johndoe"},
		Chat: models.Chat{ID: -1001234567890},
	}
	assert.True(t, IsAllowedChat(msg, []string{"-1001234567890"}))
	assert.False(t, IsAllowedChat(msg, []string{"999"}))
}

func TestIsAllowedChat_EmptyList(t *testing.T) {
	msg := &models.Message{
		From: &models.User{Username: "anyone"},
		Chat: models.Chat{ID: 1},
	}
	assert.True(t, IsAllowedChat(msg, []string{}))
}
