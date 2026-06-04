package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

const DefaultDelimiter = "\n\n***\n\n"

type DistributionRule struct {
	Filter        string `mapstructure:"filter" yaml:"filter"`
	NotePath      string `mapstructure:"note_path" yaml:"note_path"`
	FilePath      string `mapstructure:"file_path" yaml:"file_path"`
	TemplateFile  string `mapstructure:"template_file" yaml:"template_file"`
	Heading       string `mapstructure:"heading" yaml:"heading"`
	Delimiter     string `mapstructure:"delimiter" yaml:"delimiter"`
	Reaction      string `mapstructure:"reaction" yaml:"reaction"`
	ReversedOrder bool   `mapstructure:"reversed_order" yaml:"reversed_order"`
}

type Config struct {
	BotToken          string             `mapstructure:"bot_token" yaml:"bot_token"`
	AllowedChats      []string           `mapstructure:"allowed_chats" yaml:"allowed_chats"`
	DeleteMessages    bool               `mapstructure:"delete_messages" yaml:"delete_messages"`
	Reaction          string             `mapstructure:"reaction" yaml:"reaction"`
	VaultPath         string             `mapstructure:"vault_path" yaml:"vault_path"`
	DistributionRules []DistributionRule `mapstructure:"distribution_rules" yaml:"distribution_rules"`
	LogLevel          string             `mapstructure:"log_level" yaml:"log_level"`
}

// EffectiveReaction returns the reaction for a rule, falling back to the global config value.
func (c *Config) EffectiveReaction(rule *DistributionRule) string {
	if rule.Reaction != "" {
		return rule.Reaction
	}
	return c.Reaction
}

func DefaultDistributionRule() DistributionRule {
	return DistributionRule{
		Filter:    "{{all}}",
		NotePath:  "Telegram/{{content:30}} - {{messageTime:20060102150405000}}.md",
		FilePath:  "Telegram/{{file:type}}s/{{file:name}} - {{messageTime:20060102150405000}}.{{file:extension}}",
		Delimiter: DefaultDelimiter,
	}
}

func DefaultConfig() Config {
	return Config{
		VaultPath:         ".",
		LogLevel:          "info",
		DistributionRules: []DistributionRule{DefaultDistributionRule()},
	}
}

func Load(path string) (*Config, error) {
	v := viper.New()

	defaults := DefaultConfig()
	v.SetDefault("vault_path", defaults.VaultPath)
	v.SetDefault("log_level", defaults.LogLevel)

	v.SetEnvPrefix("OTS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.obsidian-telegram-sync")
		v.AddConfigPath("/etc/obsidian-telegram-sync")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	if len(cfg.DistributionRules) == 0 {
		cfg.DistributionRules = []DistributionRule{DefaultDistributionRule()}
	}
	for i := range cfg.DistributionRules {
		if cfg.DistributionRules[i].Delimiter == "" {
			cfg.DistributionRules[i].Delimiter = DefaultDelimiter
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.BotToken == "" {
		if envToken := os.Getenv("OTS_BOT_TOKEN"); envToken != "" {
			c.BotToken = envToken
		} else {
			return fmt.Errorf("bot_token is required")
		}
	}
	if c.VaultPath == "" {
		c.VaultPath = "."
	}
	return nil
}
