package markdown

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/go-telegram/bot/models"
)

// ConvertEntities converts Telegram message text with entities to Markdown.
// Telegram API uses UTF-16 offsets for entity positions.
func ConvertEntities(text string, entities []models.MessageEntity) string {
	if len(entities) == 0 {
		return text
	}

	utf16Text := utf16.Encode([]rune(text))
	copied := cloneEntities(entities)

	for i, ent := range copied {
		start := ent.Offset
		end := start + ent.Length

		if start < 0 || end > len(utf16Text) {
			continue
		}

		entityText := string(utf16.Decode(utf16Text[start:end]))
		trimmedEnd := end
		if ent.Type != models.MessageEntityTypePre {
			trimmed := strings.TrimRight(entityText, "\n")
			trimmedEnd = start + len(utf16.Encode([]rune(trimmed)))
			entityText = trimmed
		}

		before := string(utf16.Decode(utf16Text[:start]))
		entityText = string(utf16.Decode(utf16Text[start:trimmedEnd]))
		after := string(utf16.Decode(utf16Text[trimmedEnd:]))

		var beforeLen, afterLen int
		var replacement string

		switch ent.Type {
		case models.MessageEntityTypeBold:
			replacement = "**" + entityText + "**"
			beforeLen, afterLen = 2, 2
		case models.MessageEntityTypeItalic:
			replacement = "*" + entityText + "*"
			beforeLen, afterLen = 1, 1
		case models.MessageEntityTypeUnderline:
			replacement = "<u>" + entityText + "</u>"
			beforeLen, afterLen = 3, 4
		case models.MessageEntityTypeStrikethrough:
			replacement = "~~" + entityText + "~~"
			beforeLen, afterLen = 2, 2
		case models.MessageEntityTypeCode:
			replacement = "`" + entityText + "`"
			beforeLen, afterLen = 1, 1
		case models.MessageEntityTypePre:
			lang := ent.Language
			replacement = "```" + lang + "\n" + entityText + "\n```"
			beforeLen = 4 + len(utf16.Encode([]rune(lang)))
			afterLen = 4
		case models.MessageEntityTypeTextLink:
			if ent.URL != "" {
				replacement = "[" + entityText + "](" + ent.URL + ")"
				beforeLen = 1
				afterLen = 3 + len(utf16.Encode([]rune(ent.URL)))
			}
		case models.MessageEntityTypeTextMention:
			if ent.User != nil {
				link := fmt.Sprintf("tg://user?id=%d", ent.User.ID)
				replacement = "[" + entityText + "](" + link + ")"
				beforeLen = 1
				afterLen = 3 + len(utf16.Encode([]rune(link)))
			}
		case models.MessageEntityTypeSpoiler:
			replacement = "||" + entityText + "||"
			beforeLen, afterLen = 2, 2
		case models.MessageEntityTypeBlockquote, models.MessageEntityTypeExpandableBlockquote:
			lines := strings.Split(entityText, "\n")
			for j := range lines {
				lines[j] = "> " + lines[j]
			}
			replacement = strings.Join(lines, "\n")
			beforeLen = 2
			afterLen = 0
		default:
			continue
		}

		newText := before + replacement + after
		utf16Text = utf16.Encode([]rune(newText))

		updateOffsets(copied, i, ent.Offset, ent.Length, beforeLen, afterLen)
	}

	return string(utf16.Decode(utf16Text))
}

// ConvertMessage converts a message's text/caption and entities to Markdown,
// appending any inline keyboard URLs.
func ConvertMessage(text string, entities []models.MessageEntity, replyMarkup *models.InlineKeyboardMarkup) string {
	md := ConvertEntities(text, entities)
	urls := inlineURLs(replyMarkup)
	if urls != "" {
		md += "\n\n" + urls
	}
	return md
}

func inlineURLs(markup *models.InlineKeyboardMarkup) string {
	if markup == nil || len(markup.InlineKeyboard) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, row := range markup.InlineKeyboard {
		for _, btn := range row {
			if btn.URL != "" {
				sb.WriteString("[" + btn.Text + "](" + btn.URL + ")\n")
			}
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func cloneEntities(entities []models.MessageEntity) []models.MessageEntity {
	out := make([]models.MessageEntity, len(entities))
	copy(out, entities)
	return out
}

func updateOffsets(entities []models.MessageEntity, currentIdx, currentOffset, currentLength, beforeOffset, afterOffset int) {
	endOffset := currentOffset + currentLength
	for j := currentIdx + 1; j < len(entities); j++ {
		origOffset := entities[j].Offset
		if origOffset >= endOffset {
			entities[j].Offset += beforeOffset + afterOffset
		} else if origOffset >= currentOffset {
			entities[j].Offset += beforeOffset
		}
	}
}
