package telegram

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.uber.org/zap"

	"github.com/kolsha/obsidian-telegram-sync/internal/config"
	"github.com/kolsha/obsidian-telegram-sync/internal/distribution"
	"github.com/kolsha/obsidian-telegram-sync/internal/markdown"
	"github.com/kolsha/obsidian-telegram-sync/internal/storage"
)

// Bot wraps the Telegram bot and handles message processing.
type Bot struct {
	bot     *bot.Bot
	cfg     *config.Config
	logger  *zap.Logger
	botUser *models.User
}

// New creates a new Bot instance and starts polling.
func New(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Bot, error) {
	b := &Bot{
		cfg:    cfg,
		logger: logger,
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(b.handleUpdate),
	}

	tgBot, err := bot.New(cfg.BotToken, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating bot: %w", err)
	}
	b.bot = tgBot

	me, err := b.bot.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting bot info: %w", err)
	}
	b.botUser = me
	logger.Info("bot connected", zap.String("username", me.Username))

	return b, nil
}

// Start begins polling for updates. Blocks until context is cancelled.
func (b *Bot) Start(ctx context.Context) {
	b.bot.Start(ctx)
}

func (b *Bot) handleUpdate(ctx context.Context, tgBot *bot.Bot, update *models.Update) {
	var msg *models.Message

	switch {
	case update.Message != nil:
		msg = update.Message
	case update.EditedMessage != nil:
		msg = update.EditedMessage
	case update.ChannelPost != nil:
		msg = update.ChannelPost
	case update.EditedChannelPost != nil:
		msg = update.EditedChannelPost
	default:
		return
	}

	if err := b.processMessage(ctx, tgBot, msg); err != nil {
		b.logger.Error("processing message", zap.Error(err), zap.Int("message_id", msg.ID))
	}
}

func (b *Bot) processMessage(ctx context.Context, tgBot *bot.Bot, msg *models.Message) error {
	if msg.Text == "/start" {
		return nil
	}

	if !IsAllowedChat(msg, b.cfg.AllowedChats) {
		chatIDStr := strconv.FormatInt(msg.Chat.ID, 10)
		username := ""
		if msg.From != nil {
			username = msg.From.Username
		}
		replyText := fmt.Sprintf("Access denied. Add your username %q or chat id %q in the config allowed_chats.", username, chatIDStr)
		_, _ = tgBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   replyText,
			ReplyParameters: &models.ReplyParameters{
				MessageID: msg.ID,
			},
		})
		return nil
	}

	rule := distribution.MatchedRule(b.cfg.DistributionRules, msg)
	if rule == nil {
		b.logger.Debug("no matching distribution rule", zap.Int("message_id", msg.ID))
		return nil
	}

	msgCtx := &markdown.MessageContext{
		Message: msg,
		BotUser: b.botUser,
		Now:     time.Now(),
	}

	fileInfo := GetFileInfo(msg)

	var err error
	if msg.Text == "" && fileInfo != nil && rule.FilePath != "" {
		err = b.handleFile(ctx, tgBot, msg, rule, msgCtx, fileInfo)
	} else {
		err = b.handleText(ctx, msg, rule, msgCtx)
	}
	if err != nil {
		return err
	}

	b.setReaction(ctx, tgBot, msg, rule)
	return nil
}

func (b *Bot) handleText(_ context.Context, msg *models.Message, rule *config.DistributionRule, msgCtx *markdown.MessageContext) error {
	content := markdown.RenderNoteContent(msgCtx, b.readTemplate(rule.TemplateFile))
	notePath := markdown.RenderNotePath(msgCtx, rule.NotePath)

	if notePath == "" {
		return nil
	}

	fullNotePath := filepath.Join(b.cfg.VaultPath, notePath)
	noteDir := filepath.Dir(fullNotePath)
	if err := storage.CreateDirIfNotExist(noteDir); err != nil {
		return fmt.Errorf("creating note directory: %w", err)
	}

	if err := storage.AppendContentToNote(fullNotePath, content, rule.Heading, rule.Delimiter, rule.ReversedOrder); err != nil {
		return fmt.Errorf("writing note: %w", err)
	}

	b.logger.Info("note written",
		zap.String("path", notePath),
		zap.Int("message_id", msg.ID),
	)

	return nil
}

func (b *Bot) handleFile(ctx context.Context, tgBot *bot.Bot, msg *models.Message, rule *config.DistributionRule, msgCtx *markdown.MessageContext, fileInfo *FileInfo) error {
	file, err := tgBot.GetFile(ctx, &bot.GetFileParams{FileID: fileInfo.FileID})
	if err != nil {
		return fmt.Errorf("getting file info: %w", err)
	}

	fileVars := markdown.FileVars{
		Type:      fileInfo.Type,
		Name:      fileInfo.BaseName(),
		Extension: fileInfo.FileExtension(file.FilePath),
		UniqueID:  fileInfo.UniqueID,
	}

	filePath := markdown.RenderFilePath(msgCtx, rule.FilePath, fileVars)
	if filePath == "" {
		return nil
	}

	fullFilePath := filepath.Join(b.cfg.VaultPath, filePath)
	fileDir := filepath.Dir(fullFilePath)
	if err := storage.CreateDirIfNotExist(fileDir); err != nil {
		return fmt.Errorf("creating file directory: %w", err)
	}

	fullFilePath = storage.GetUniqueFilePath(fullFilePath, time.Now(), fileVars.Extension, nil)

	if err := storage.DownloadFile(ctx, tgBot, file, fullFilePath); err != nil {
		return fmt.Errorf("downloading file: %w", err)
	}

	relPath, err := filepath.Rel(b.cfg.VaultPath, fullFilePath)
	if err != nil {
		return fmt.Errorf("resolving saved file path: %w", err)
	}
	fileVars.Path = filepath.ToSlash(relPath)

	b.logger.Info("file saved",
		zap.String("path", fileVars.Path),
		zap.String("type", fileInfo.Type),
		zap.Int("message_id", msg.ID),
	)

	if msg.Caption != "" || rule.TemplateFile != "" {
		msgCtx.FilesLinks = []string{markdown.RenderFileLink(rule.FileLinkTemplate, fileVars)}

		notePath := markdown.RenderNotePath(msgCtx, rule.NotePath)
		if notePath != "" {
			content := markdown.RenderNoteContent(msgCtx, b.readTemplate(rule.TemplateFile))
			fullNotePath := filepath.Join(b.cfg.VaultPath, notePath)
			noteDir := filepath.Dir(fullNotePath)
			if err := storage.CreateDirIfNotExist(noteDir); err != nil {
				return fmt.Errorf("creating note directory: %w", err)
			}

			if err := storage.AppendContentToNote(fullNotePath, content, rule.Heading, rule.Delimiter, rule.ReversedOrder); err != nil {
				return fmt.Errorf("writing note for file: %w", err)
			}
		}
	}

	return nil
}

func (b *Bot) setReaction(ctx context.Context, tgBot *bot.Bot, msg *models.Message, rule *config.DistributionRule) {
	emoji := b.cfg.EffectiveReaction(rule)
	if emoji == "" {
		return
	}
	_, err := tgBot.SetMessageReaction(ctx, &bot.SetMessageReactionParams{
		ChatID:    msg.Chat.ID,
		MessageID: msg.ID,
		Reaction: []models.ReactionType{{
			Type:              models.ReactionTypeTypeEmoji,
			ReactionTypeEmoji: &models.ReactionTypeEmoji{Emoji: emoji},
		}},
	})
	if err != nil {
		b.logger.Debug("failed to set reaction", zap.Error(err), zap.Int("message_id", msg.ID))
	}
}

func (b *Bot) readTemplate(templatePath string) string {
	if templatePath == "" {
		return ""
	}
	fullPath := filepath.Join(b.cfg.VaultPath, templatePath)
	data, err := storage.ReadFile(fullPath)
	if err != nil {
		b.logger.Warn("template file not found", zap.String("path", templatePath), zap.Error(err))
		return ""
	}
	return string(data)
}
