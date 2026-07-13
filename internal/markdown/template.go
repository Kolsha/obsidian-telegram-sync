package markdown

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/go-telegram/bot/models"
)

// MessageContext holds all data needed for template variable substitution.
type MessageContext struct {
	Message    *models.Message
	BotUser    *models.User
	FilesLinks []string
	Now        time.Time
}

func formatDateTime(dt time.Time, layout string) string {
	return dt.Format(layout)
}

func unixToTime(unixTime int) time.Time {
	return time.Unix(int64(unixTime), 0)
}

func getForwardDate(msg *models.Message) int {
	if msg.ForwardOrigin == nil {
		return 0
	}
	switch msg.ForwardOrigin.Type {
	case models.MessageOriginTypeUser:
		if msg.ForwardOrigin.MessageOriginUser != nil {
			return msg.ForwardOrigin.MessageOriginUser.Date
		}
	case models.MessageOriginTypeHiddenUser:
		if msg.ForwardOrigin.MessageOriginHiddenUser != nil {
			return msg.ForwardOrigin.MessageOriginHiddenUser.Date
		}
	case models.MessageOriginTypeChat:
		if msg.ForwardOrigin.MessageOriginChat != nil {
			return msg.ForwardOrigin.MessageOriginChat.Date
		}
	case models.MessageOriginTypeChannel:
		if msg.ForwardOrigin.MessageOriginChannel != nil {
			return msg.ForwardOrigin.MessageOriginChannel.Date
		}
	}
	return 0
}

func getForwardFromName(msg *models.Message) string {
	if msg.ForwardOrigin == nil {
		return ""
	}
	switch msg.ForwardOrigin.Type {
	case models.MessageOriginTypeUser:
		u := msg.ForwardOrigin.MessageOriginUser
		if u == nil {
			return ""
		}
		name := u.SenderUser.FirstName
		if u.SenderUser.LastName != "" {
			name += " " + u.SenderUser.LastName
		}
		return name
	case models.MessageOriginTypeHiddenUser:
		u := msg.ForwardOrigin.MessageOriginHiddenUser
		if u == nil {
			return ""
		}
		return u.SenderUserName
	case models.MessageOriginTypeChat:
		c := msg.ForwardOrigin.MessageOriginChat
		if c == nil {
			return ""
		}
		name := c.SenderChat.Title
		if c.AuthorSignature != nil && *c.AuthorSignature != "" {
			name += "(" + *c.AuthorSignature + ")"
		}
		if name == "" {
			name = c.SenderChat.Username
		}
		return name
	case models.MessageOriginTypeChannel:
		c := msg.ForwardOrigin.MessageOriginChannel
		if c == nil {
			return ""
		}
		name := c.Chat.Title
		if c.AuthorSignature != nil && *c.AuthorSignature != "" {
			name += "(" + *c.AuthorSignature + ")"
		}
		if name == "" {
			name = c.Chat.Username
		}
		return name
	}
	return ""
}

func getForwardFromLink(msg *models.Message) string {
	name := getForwardFromName(msg)
	if name == "" {
		return ""
	}

	var username string
	switch msg.ForwardOrigin.Type {
	case models.MessageOriginTypeUser:
		u := msg.ForwardOrigin.MessageOriginUser.SenderUser
		username = u.Username
		if username == "" {
			username = fmt.Sprintf("no_username_%d", u.ID)
		}
	case models.MessageOriginTypeHiddenUser:
		d := msg.ForwardOrigin.MessageOriginHiddenUser.Date
		username = fmt.Sprintf("hidden_account_%d", d)
	case models.MessageOriginTypeChat:
		c := msg.ForwardOrigin.MessageOriginChat.SenderChat
		username = c.Username
		if username == "" {
			username = chatIDPath(c.ID)
		}
	case models.MessageOriginTypeChannel:
		ch := msg.ForwardOrigin.MessageOriginChannel
		username = ch.Chat.Username
		if username == "" {
			username = chatIDPath(ch.Chat.ID)
		}
		if ch.MessageID != 0 {
			username += "/" + strconv.Itoa(ch.MessageID)
		}
	}

	return "[" + name + "](https://t.me/" + username + ")"
}

func chatIDPath(id int64) string {
	s := strconv.FormatInt(id, 10)
	if id < 0 && len(s) > 4 {
		return "c/" + s[4:]
	}
	return "c/" + s
}

func getUserLink(msg *models.Message) string {
	if msg.From == nil {
		return ""
	}
	username := msg.From.Username
	if username == "" {
		username = fmt.Sprintf("no_username_%d", msg.From.ID)
	}
	fullName := msg.From.FirstName
	if msg.From.LastName != "" {
		fullName += " " + msg.From.LastName
	}
	return "[" + fullName + "](https://t.me/" + username + ")"
}

func getChatName(msg *models.Message, botUser *models.User) string {
	if botUser != nil && msg.From != nil && msg.Chat.ID == msg.From.ID {
		name := botUser.FirstName
		if botUser.LastName != "" {
			name += " " + botUser.LastName
		}
		return strings.TrimSpace(name)
	}
	if msg.Chat.Type == "private" {
		name := msg.Chat.FirstName
		if msg.Chat.LastName != "" {
			name += " " + msg.Chat.LastName
		}
		return strings.TrimSpace(name)
	}
	if msg.Chat.Title != "" {
		return msg.Chat.Title
	}
	return string(msg.Chat.Type) + strconv.FormatInt(msg.Chat.ID, 10)
}

func getChatLink(msg *models.Message, botUser *models.User) string {
	chatName := getChatName(msg, botUser)
	var userName string
	if botUser != nil && msg.From != nil && msg.Chat.ID == msg.From.ID {
		userName = botUser.Username
	} else if msg.Chat.Type == "private" {
		userName = msg.Chat.Username
		if userName == "" {
			userName = fmt.Sprintf("no_username_%d", msg.Chat.ID)
		}
	} else {
		userName = msg.Chat.Username
		if userName == "" {
			s := strconv.FormatInt(msg.Chat.ID, 10)
			if msg.Chat.ID < 0 && len(s) > 4 {
				userName = "c/" + s[4:]
			} else {
				userName = "c/" + s
			}
		}
	}
	return "[" + chatName + "](https://t.me/" + userName + ")"
}

func getChatID(msg *models.Message, botUser *models.User) string {
	if botUser != nil && msg.From != nil && msg.Chat.ID == msg.From.ID {
		return strconv.FormatInt(botUser.ID, 10)
	}
	return strconv.FormatInt(msg.Chat.ID, 10)
}

func getFirstURL(msg *models.Message) string {
	text := msg.Text + msg.Caption
	if text == "" {
		return ""
	}
	re := regexp.MustCompile(`https?://[^\s<>\[\]()]+`)
	return re.FindString(text)
}

func getHashtag(msg *models.Message, num int) string {
	text := msg.Text + msg.Caption
	if text == "" {
		return ""
	}
	re := regexp.MustCompile(`#[\p{L}\p{N}_]+`)
	matches := re.FindAllString(text, -1)
	if num < 1 || num > len(matches) {
		return ""
	}
	return strings.TrimPrefix(matches[num-1], "#")
}

func getReplyMessageID(msg *models.Message) string {
	if msg.ReplyToMessage != nil {
		if msg.ReplyToMessage.MessageThreadID != msg.ReplyToMessage.ID {
			return strconv.Itoa(msg.ReplyToMessage.ID)
		}
	}
	return ""
}

func getTopicID(msg *models.Message) string {
	if msg.IsTopicMessage {
		if msg.MessageThreadID != 0 {
			return strconv.Itoa(msg.MessageThreadID)
		}
		return "1"
	}
	return ""
}

func processContent(text string, property string) string {
	if property == "" || strings.EqualFold(property, "text") {
		return text
	}
	if n, err := strconv.Atoi(property); err == nil && n > 0 {
		runes := []rune(text)
		if n > len(runes) {
			n = len(runes)
		}
		return string(runes[:n])
	}

	lines := strings.Split(text, "\n")

	rangeRE := regexp.MustCompile(`^\[(\d+)-(\d+)\]$`)
	if m := rangeRE.FindStringSubmatch(property); m != nil {
		start, _ := strconv.Atoi(m[1])
		end, _ := strconv.Atoi(m[2])
		start = max(0, start-1)
		end = min(len(lines), end)
		return strings.Join(lines[start:end], "\n")
	}

	singleRE := regexp.MustCompile(`^\[(\d+)\]$`)
	if m := singleRE.FindStringSubmatch(property); m != nil {
		idx, _ := strconv.Atoi(m[1])
		idx--
		if idx >= 0 && idx < len(lines) {
			return lines[idx]
		}
		return ""
	}

	lastRE := regexp.MustCompile(`^\[-(\d+)\]$`)
	if m := lastRE.FindStringSubmatch(property); m != nil {
		n, _ := strconv.Atoi(m[1])
		idx := max(0, len(lines)-n-1)
		if idx < len(lines) {
			return lines[idx]
		}
		return ""
	}

	fromRE := regexp.MustCompile(`^\[(\d+)-\]$`)
	if m := fromRE.FindStringSubmatch(property); m != nil {
		start, _ := strconv.Atoi(m[1])
		start = max(0, start-1)
		return strings.Join(lines[start:], "\n")
	}

	return ""
}

// SanitizeForPath removes characters invalid in file paths.
func SanitizeForPath(s string) string {
	return regexp.MustCompile(`[\\:*?"<>|\n\r]`).ReplaceAllString(s, "_")
}

// SanitizeFileName removes characters invalid in file names (includes /).
func SanitizeFileName(s string) string {
	return regexp.MustCompile(`[\\/:*?"<>|\n\r]`).ReplaceAllString(s, "_")
}

// RenderNoteContent applies template variables to the note content template.
func RenderNoteContent(ctx *MessageContext, templateContent string) string {
	msg := ctx.Message
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}
	entities := msg.Entities
	if len(entities) == 0 {
		entities = msg.CaptionEntities
	}

	allEmbedded := strings.Join(ctx.FilesLinks, "\n")
	allLinks := strings.ReplaceAll(allEmbedded, "![", "[")

	var textMD string
	if templateContent == "" || strings.Contains(templateContent, "{{content") {
		textMD = ConvertMessage(text, entities, msg.ReplyMarkup)
	}

	forwardLink := getForwardFromLink(msg)
	fullContent := ""
	if forwardLink != "" {
		fullContent += "**Forwarded from " + forwardLink + "**\n\n"
	}
	if allEmbedded != "" {
		fullContent += allEmbedded + "\n\n"
	}
	fullContent += textMD

	if templateContent == "" {
		return fullContent
	}

	result := renderBasicVars(ctx, templateContent, textMD, fullContent, false)
	result = strings.ReplaceAll(result, "{{files}}", allEmbedded)
	result = strings.ReplaceAll(result, "{{files:links}}", allLinks)
	result = strings.ReplaceAll(result, "{{url1}}", getFirstURL(msg))

	var rules []replacementRule
	result, rules = parseReplacementDirectives(result)
	result = applyReplacementRules(result, rules)

	return result
}

// RenderNotePath applies template variables to a note path template.
func RenderNotePath(ctx *MessageContext, pathTemplate string) string {
	if pathTemplate == "" {
		return ""
	}
	result := pathTemplate
	if strings.HasSuffix(result, "/") {
		result += "{{content:30}} - {{messageTime:20060102150405000}}.md"
	}

	textContent := ctx.Message.Text
	if textContent == "" {
		textContent = ctx.Message.Caption
	}

	result = renderBasicVars(ctx, result, textContent, textContent, true)

	if strings.HasSuffix(result, "/.md") {
		result = strings.TrimSuffix(result, "/.md") + "/_.md"
	}

	lastSlash := strings.LastIndex(result, "/")
	afterSlash := result
	if lastSlash >= 0 {
		afterSlash = result[lastSlash:]
	}
	if !strings.Contains(afterSlash, ".") {
		result += ".md"
	}
	if strings.HasSuffix(result, ".") {
		result += "md"
	}

	return SanitizeForPath(result)
}

// FileVars holds file attributes available to file path and file link templates.
type FileVars struct {
	Type      string
	Name      string
	Extension string
	UniqueID  string
	// Path is the vault-relative path of the saved file. Empty during path
	// rendering; set before link rendering.
	Path string
}

func replaceFileVars(s string, vars FileVars) string {
	s = strings.ReplaceAll(s, "{{file:type}}", vars.Type)
	s = strings.ReplaceAll(s, "{{file:name}}", vars.Name)
	s = strings.ReplaceAll(s, "{{file:extension}}", vars.Extension)
	s = strings.ReplaceAll(s, "{{file:uniqueId}}", vars.UniqueID)
	s = strings.ReplaceAll(s, "{{file:path}}", vars.Path)
	return s
}

// RenderFilePath applies template variables to a file path template.
func RenderFilePath(ctx *MessageContext, pathTemplate string, vars FileVars) string {
	if pathTemplate == "" {
		return ""
	}
	result := pathTemplate
	if strings.HasSuffix(result, "/") {
		result += "{{file:name}} - {{messageTime:20060102150405000}}.{{file:extension}}"
	}

	result = renderBasicVars(ctx, result, ctx.Message.Caption, ctx.Message.Caption, true)
	result = replaceFileVars(result, vars)

	lastSlash := strings.LastIndex(result, "/")
	afterSlash := result
	if lastSlash >= 0 {
		afterSlash = result[lastSlash+1:]
	}
	if !strings.Contains(afterSlash, ".") {
		result += "." + vars.Extension
	}
	if strings.HasSuffix(result, ".") {
		result += vars.Extension
	}

	return SanitizeForPath(result)
}

// RenderFileLink renders the markdown link for a saved file using a link template.
func RenderFileLink(linkTemplate string, vars FileVars) string {
	return replaceFileVars(linkTemplate, vars)
}

func renderBasicVars(ctx *MessageContext, template, messageText, messageContent string, isPath bool) string {
	msg := ctx.Message
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}
	msgTime := unixToTime(msg.Date)
	creationTime := msgTime
	if fwd := getForwardDate(msg); fwd != 0 {
		creationTime = unixToTime(fwd)
	}

	sanitize := func(s string) string {
		if isPath {
			return SanitizeFileName(s)
		}
		return s
	}

	result := template

	contentRE := regexp.MustCompile(`\{\{content(?::([^}]*))?\}\}`)
	result = contentRE.ReplaceAllStringFunc(result, func(match string) string {
		parts := contentRE.FindStringSubmatch(match)
		prop := ""
		if len(parts) > 1 {
			prop = parts[1]
		}
		src := messageContent
		if prop != "" {
			src = messageText
		}
		if src == "" {
			src = messageText
		}
		return sanitize(processContent(src, prop))
	})

	dateTimeRE := regexp.MustCompile(`\{\{(messageDate|messageTime|date|time|creationDate|creationTime):(.*?)\}\}`)
	result = dateTimeRE.ReplaceAllStringFunc(result, func(match string) string {
		parts := dateTimeRE.FindStringSubmatch(match)
		varName, format := parts[1], parts[2]
		var dt time.Time
		switch varName {
		case "messageDate", "messageTime":
			dt = msgTime
		case "date", "time":
			dt = now
		case "creationDate", "creationTime":
			dt = creationTime
		}
		return formatDateTime(dt, format)
	})

	result = strings.ReplaceAll(result, "{{forwardFrom}}", getForwardFromLink(msg))
	result = strings.ReplaceAll(result, "{{forwardFrom:name}}", sanitize(getForwardFromName(msg)))
	result = strings.ReplaceAll(result, "{{user}}", getUserLink(msg))

	if msg.From != nil {
		result = strings.ReplaceAll(result, "{{user:name}}", sanitize(msg.From.Username))
		fullName := msg.From.FirstName
		if msg.From.LastName != "" {
			fullName += " " + msg.From.LastName
		}
		result = strings.ReplaceAll(result, "{{user:fullName}}", sanitize(strings.TrimSpace(fullName)))
		result = strings.ReplaceAll(result, "{{userId}}", strconv.FormatInt(msg.From.ID, 10))
	} else {
		result = strings.ReplaceAll(result, "{{user:name}}", "")
		result = strings.ReplaceAll(result, "{{user:fullName}}", "")
		result = strings.ReplaceAll(result, "{{userId}}", "")
	}

	result = strings.ReplaceAll(result, "{{chat}}", getChatLink(msg, ctx.BotUser))
	result = strings.ReplaceAll(result, "{{chatId}}", getChatID(msg, ctx.BotUser))
	result = strings.ReplaceAll(result, "{{chat:name}}", sanitize(getChatName(msg, ctx.BotUser)))
	result = strings.ReplaceAll(result, "{{topicId}}", getTopicID(msg))
	result = strings.ReplaceAll(result, "{{messageId}}", strconv.Itoa(msg.ID))
	result = strings.ReplaceAll(result, "{{replyMessageId}}", getReplyMessageID(msg))

	hashtagRE := regexp.MustCompile(`\{\{hashtag:\[(\d+)\]\}\}`)
	result = hashtagRE.ReplaceAllStringFunc(result, func(match string) string {
		parts := hashtagRE.FindStringSubmatch(match)
		n, _ := strconv.Atoi(parts[1])
		return getHashtag(msg, n)
	})

	voiceRE := regexp.MustCompile(`\{\{voiceTranscript(?::[^}]*)?\}\}`)
	result = voiceRE.ReplaceAllString(result, "")

	return result
}

// UTF16Len returns the number of UTF-16 code units for a string.
func UTF16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}
