package markdown

import (
	"testing"
	"time"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
)

func testMsg() *models.Message {
	return &models.Message{
		ID:   42,
		Date: 1700000000,
		From: &models.User{
			ID:        12345,
			FirstName: "John",
			LastName:  "Doe",
			Username:  "johndoe",
		},
		Chat: models.Chat{
			ID:    67890,
			Type:  "private",
			Title: "",
			FirstName: "John",
			LastName:  "Doe",
			Username: "johndoe",
		},
		Text: "Hello world! Check #golang and #testing",
	}
}

func testCtx(msg *models.Message) *MessageContext {
	return &MessageContext{
		Message: msg,
		Now:     time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
	}
}

func TestProcessContent(t *testing.T) {
	text := "line1\nline2\nline3\nline4\nline5"

	tests := []struct {
		name string
		prop string
		want string
	}{
		{"full text", "", text},
		{"first 10 chars", "10", "line1\nline"},
		{"line range [2-4]", "[2-4]", "line2\nline3\nline4"},
		{"single line [1]", "[1]", "line1"},
		{"from line to end [3-]", "[3-]", "line3\nline4\nline5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, processContent(text, tt.prop))
		})
	}
}

func TestRenderNoteContent_NoTemplate(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	content := RenderNoteContent(ctx, "")
	assert.Equal(t, "Hello world! Check #golang and #testing", content)
}

func TestRenderNoteContent_WithTemplate(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "# {{user:fullName}}\n\n{{content}}\n\nSent: {{messageDate:2006-01-02}}"
	content := RenderNoteContent(ctx, tpl)

	assert.Contains(t, content, "# John Doe")
	assert.Contains(t, content, "Hello world! Check #golang and #testing")
	assert.Contains(t, content, "Sent: 2023-11-14")
}

func TestRenderNoteContent_Forward(t *testing.T) {
	msg := testMsg()
	msg.ForwardOrigin = &models.MessageOrigin{
		Type: models.MessageOriginTypeUser,
		MessageOriginUser: &models.MessageOriginUser{
			Date: 1699999000,
			SenderUser: models.User{
				ID:        999,
				FirstName: "Alice",
				Username:  "alice",
			},
		},
	}
	ctx := testCtx(msg)

	content := RenderNoteContent(ctx, "")
	assert.Contains(t, content, "**Forwarded from [Alice](https://t.me/alice)**")
}

func TestRenderNoteContent_WithFiles(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)
	ctx.FilesLinks = []string{"![photo](Telegram/photos/img.jpg)"}

	content := RenderNoteContent(ctx, "")
	assert.Contains(t, content, "![photo](Telegram/photos/img.jpg)")
}

func TestRenderNoteContent_Replace(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "{{content}}\n{{replace:world=>Go}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Contains(t, content, "Hello Go!")
}

func TestRenderNoteContent_ReplaceRe(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		tpl         string
		wantContain string
	}{
		{
			name:        "wildcard match",
			text:        "start middle end",
			tpl:         "{{content}}\n{{replaceRe:start.*end=>fixed}}",
			wantContain: "fixed",
		},
		{
			name:        "capture group",
			text:        "Hello World",
			tpl:         `{{content}}` + "\n" + `{{replaceRe:Hello (\w+)=>Hi $1}}`,
			wantContain: "Hi World",
		},
		{
			name:        "delete regex match",
			text:        "foo123bar",
			tpl:         `{{content}}` + "\n" + `{{replaceRe:\d+}}`,
			wantContain: "foobar",
		},
		{
			name:        "regex and literal together",
			text:        "Hello world 123",
			tpl:         "{{content}}\n{{replaceRe:\\d+=>NUM}}{{replace:Hello=>Hi}}",
			wantContain: "Hi world NUM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := testMsg()
			msg.Text = tt.text
			ctx := testCtx(msg)
			content := RenderNoteContent(ctx, tt.tpl)
			assert.Contains(t, content, tt.wantContain)
		})
	}
}

func TestRenderNotePath(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	path := RenderNotePath(ctx, "Telegram/{{content:10}} - {{messageTime:20060102}}.md")
	assert.Contains(t, path, "Telegram/")
	assert.Contains(t, path, "Hello worl")
	assert.Contains(t, path, "20231114")
	assert.True(t, len(path) > 0)
}

func TestRenderNotePath_TrailingSlash(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	path := RenderNotePath(ctx, "Inbox/")
	assert.Contains(t, path, "Inbox/")
	assert.Contains(t, path, ".md")
}

func TestRenderFilePath(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	path := RenderFilePath(ctx, "Files/{{file:type}}s/{{file:name}}.{{file:extension}}", "photo", "sunset", "jpg")
	assert.Equal(t, "Files/photos/sunset.jpg", path)
}

func TestRenderFilePath_TrailingSlash(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	path := RenderFilePath(ctx, "Files/", "photo", "sunset", "jpg")
	assert.Contains(t, path, "Files/")
	assert.Contains(t, path, "sunset")
	assert.Contains(t, path, ".jpg")
}

func TestTemplateVars_User(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "{{user:name}} {{user:fullName}} {{userId}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Contains(t, content, "johndoe")
	assert.Contains(t, content, "John Doe")
	assert.Contains(t, content, "12345")
}

func TestTemplateVars_UserNilFrom(t *testing.T) {
	msg := testMsg()
	msg.From = nil
	ctx := testCtx(msg)

	tpl := "user={{userId}} msg={{messageId}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Equal(t, "user= msg=42", content)
}

func TestTemplateVars_Chat(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "{{chat:name}} {{chatId}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Contains(t, content, "John Doe")
	assert.Contains(t, content, "67890")
}

func TestTemplateVars_MessageId(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "msg={{messageId}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Equal(t, "msg=42", content)
}

func TestTemplateVars_Hashtag(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "{{hashtag:[1]}} {{hashtag:[2]}} {{hashtag:[3]}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Contains(t, content, "golang")
	assert.Contains(t, content, "testing")
}

func TestTemplateVars_DateFormats(t *testing.T) {
	msg := testMsg()
	ctx := testCtx(msg)

	tpl := "{{date:2006-01-02}} {{time:15:04}}"
	content := RenderNoteContent(ctx, tpl)
	assert.Contains(t, content, "2024-01-15")
	assert.Contains(t, content, "10:30")
}

func TestRenderNoteContent_MessageTimeInsideReplace(t *testing.T) {
	msg := testMsg()
	msg.Text = "Вам куда?"
	msg.ForwardOrigin = &models.MessageOrigin{
		Type: models.MessageOriginTypeUser,
		MessageOriginUser: &models.MessageOriginUser{
			Date: 1699999000,
			SenderUser: models.User{
				ID:        999,
				FirstName: "Коля",
				Username:  "Kolsha",
			},
		},
	}
	ctx := testCtx(msg)

	tpl := "\n{{content}}\n{{replace:**Forwarded from [Коля](https://t.me/Kolsha)**=>---\\n{{messageTime:15:04}}:}}{{replace:\\n\\n=>\\n}}{{replace:#todo=>}}\n"
	content := RenderNoteContent(ctx, tpl)

	t.Logf("rendered content:\n%s", content)
	assert.Contains(t, content, "---")
	assert.NotContains(t, content, "{{messageTime")
	assert.NotContains(t, content, "Forwarded from")
	assert.Contains(t, content, "Вам куда?")
}

func TestSanitizeFileName(t *testing.T) {
	assert.Equal(t, "hello_world", SanitizeFileName("hello:world"))
	assert.Equal(t, "a_b_c", SanitizeFileName("a\\b/c"))
	assert.Equal(t, "test_file", SanitizeFileName("test*file"))
}

func TestGetFirstURL(t *testing.T) {
	msg := &models.Message{Text: "Check https://example.com and more"}
	assert.Equal(t, "https://example.com", getFirstURL(msg))

	msg2 := &models.Message{Text: "no urls here"}
	assert.Equal(t, "", getFirstURL(msg2))
}
