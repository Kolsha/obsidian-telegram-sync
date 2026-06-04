package markdown

import (
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
)

func entity(typ models.MessageEntityType, offset, length int) models.MessageEntity {
	return models.MessageEntity{Type: typ, Offset: offset, Length: length}
}

func TestConvertEntities(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		entities []models.MessageEntity
		want     string
	}{
		{
			name: "plain text",
			text: "hello world",
			want: "hello world",
		},
		{
			name:     "bold",
			text:     "hello world",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeBold, 0, 5)},
			want:     "**hello** world",
		},
		{
			name:     "italic",
			text:     "hello world",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeItalic, 6, 5)},
			want:     "hello *world*",
		},
		{
			name:     "underline",
			text:     "underlined text",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeUnderline, 0, 10)},
			want:     "<u>underlined</u> text",
		},
		{
			name:     "strikethrough",
			text:     "deleted text",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeStrikethrough, 0, 7)},
			want:     "~~deleted~~ text",
		},
		{
			name:     "inline code",
			text:     "use fmt.Println here",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeCode, 4, 11)},
			want:     "use `fmt.Println` here",
		},
		{
			name:     "pre block",
			text:     "code:\nfmt.Println()",
			entities: []models.MessageEntity{entity(models.MessageEntityTypePre, 6, 13)},
			want:     "code:\n```\nfmt.Println()\n```",
		},
		{
			name: "pre block with language",
			text: "fmt.Println()",
			entities: []models.MessageEntity{{
				Type:     models.MessageEntityTypePre,
				Offset:   0,
				Length:   13,
				Language: "go",
			}},
			want: "```go\nfmt.Println()\n```",
		},
		{
			name: "text link",
			text: "click here for info",
			entities: []models.MessageEntity{{
				Type:   models.MessageEntityTypeTextLink,
				Offset: 6, Length: 4,
				URL: "https://example.com",
			}},
			want: "click [here](https://example.com) for info",
		},
		{
			name: "text mention",
			text: "hello John",
			entities: []models.MessageEntity{{
				Type:   models.MessageEntityTypeTextMention,
				Offset: 6, Length: 4,
				User: &models.User{ID: 12345},
			}},
			want: "hello [John](tg://user?id=12345)",
		},
		{
			name:     "spoiler",
			text:     "the answer is 42",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeSpoiler, 14, 2)},
			want:     "the answer is ||42||",
		},
		{
			name:     "blockquote",
			text:     "quote:\nline1\nline2",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeBlockquote, 7, 11)},
			want:     "quote:\n> line1\n> line2",
		},
		{
			name: "multiple entities",
			text: "bold and italic text",
			entities: []models.MessageEntity{
				entity(models.MessageEntityTypeBold, 0, 4),
				entity(models.MessageEntityTypeItalic, 9, 6),
			},
			want: "**bold** and *italic* text",
		},
		{
			name:     "emoji (UTF-16 surrogate pair)",
			text:     "👍 bold text",
			entities: []models.MessageEntity{entity(models.MessageEntityTypeBold, 3, 4)},
			want:     "👍 **bold** text",
		},
		{
			name: "url entity is passthrough",
			text: "visit https://example.com today",
			entities: []models.MessageEntity{
				entity(models.MessageEntityTypeURL, 6, 19),
			},
			want: "visit https://example.com today",
		},
		{
			name: "link then bold (asymmetric before/after offsets)",
			text: "click here then bold end",
			entities: []models.MessageEntity{
				{Type: models.MessageEntityTypeTextLink, Offset: 6, Length: 4, URL: "https://x.co"},
				entity(models.MessageEntityTypeBold, 16, 4),
			},
			want: "click [here](https://x.co) then **bold** end",
		},
		{
			name: "three sequential entities",
			text: "aaa bbb ccc ddd",
			entities: []models.MessageEntity{
				entity(models.MessageEntityTypeBold, 0, 3),
				entity(models.MessageEntityTypeItalic, 4, 3),
				entity(models.MessageEntityTypeCode, 8, 3),
			},
			want: "**aaa** *bbb* `ccc` ddd",
		},
		{
			name: "nested: underline wrapping bold near end",
			text: "AB",
			entities: []models.MessageEntity{
				entity(models.MessageEntityTypeUnderline, 0, 2),
				entity(models.MessageEntityTypeBold, 1, 1),
			},
			want: "<u>A**B**</u>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertEntities(tt.text, tt.entities)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConvertMessage_InlineKeyboard(t *testing.T) {
	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Google", URL: "https://google.com"},
				{Text: "GitHub", URL: "https://github.com"},
			},
		},
	}

	got := ConvertMessage("check links", nil, markup)
	assert.Equal(t, "check links\n\n[Google](https://google.com)\n[GitHub](https://github.com)", got)
}

func TestConvertMessage_NoMarkup(t *testing.T) {
	got := ConvertMessage("plain text", nil, nil)
	assert.Equal(t, "plain text", got)
}
