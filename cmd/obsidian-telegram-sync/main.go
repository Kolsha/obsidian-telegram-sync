package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/kolsha/obsidian-telegram-sync/internal/config"
	"github.com/kolsha/obsidian-telegram-sync/internal/telegram"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "", "path to config file (default: ./config.yaml)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("obsidian-telegram-sync", version)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	defer logger.Sync()

	logger.Info("starting obsidian-telegram-sync",
		zap.String("version", version),
		zap.String("vault_path", cfg.VaultPath),
		zap.Int("rules", len(cfg.DistributionRules)),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	bot, err := telegram.New(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("failed to create bot", zap.Error(err))
	}

	logger.Info("bot started, polling for messages...")
	bot.Start(ctx)

	logger.Info("shutting down gracefully")
}

func newLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapLevel)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := cfg.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	return logger
}
