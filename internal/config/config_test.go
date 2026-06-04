package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, ".", cfg.VaultPath)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Len(t, cfg.DistributionRules, 1)
	assert.Equal(t, "{{all}}", cfg.DistributionRules[0].Filter)
	assert.Equal(t, DefaultDelimiter, cfg.DistributionRules[0].Delimiter)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
bot_token: "123456:ABC"
allowed_chats:
  - "testuser"
  - "-100123"
delete_messages: true
vault_path: "/tmp/notes"
log_level: "debug"
distribution_rules:
  - filter: "{{all}}"
    note_path: "Notes/{{content:20}}.md"
    file_path: "Files/{{file:name}}.{{file:extension}}"
    heading: "## Messages"
    delimiter: "\n---\n"
    reversed_order: true
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, "123456:ABC", cfg.BotToken)
	assert.Equal(t, []string{"testuser", "-100123"}, cfg.AllowedChats)
	assert.True(t, cfg.DeleteMessages)
	assert.Equal(t, "/tmp/notes", cfg.VaultPath)
	assert.Equal(t, "debug", cfg.LogLevel)

	require.Len(t, cfg.DistributionRules, 1)
	rule := cfg.DistributionRules[0]
	assert.Equal(t, "{{all}}", rule.Filter)
	assert.Equal(t, "Notes/{{content:20}}.md", rule.NotePath)
	assert.Equal(t, "Files/{{file:name}}.{{file:extension}}", rule.FilePath)
	assert.Equal(t, "## Messages", rule.Heading)
	assert.Equal(t, "\n---\n", rule.Delimiter)
	assert.True(t, rule.ReversedOrder)
}

func TestLoadMissingToken(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `vault_path: "/tmp/notes"`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	os.Unsetenv("OTS_BOT_TOKEN")
	_, err := Load(cfgPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bot_token is required")
}

func TestLoadTokenFromEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `vault_path: "/tmp/notes"`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	t.Setenv("OTS_BOT_TOKEN", "env-token-123")

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "env-token-123", cfg.BotToken)
}

func TestLoadDefaultRules(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `bot_token: "123:ABC"`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.Len(t, cfg.DistributionRules, 1)
	assert.Equal(t, "{{all}}", cfg.DistributionRules[0].Filter)
}

func TestLoadMultipleRules(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	yaml := `
bot_token: "123:ABC"
distribution_rules:
  - filter: "{{chat=Work}}"
    note_path: "Work/{{content:30}}.md"
    file_path: "Work/files/{{file:name}}.{{file:extension}}"
  - filter: "{{all}}"
    note_path: "Inbox/{{content:30}}.md"
    file_path: "Inbox/files/{{file:name}}.{{file:extension}}"
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(yaml), 0644))

	cfg, err := Load(cfgPath)
	require.NoError(t, err)
	require.Len(t, cfg.DistributionRules, 2)
	assert.Equal(t, "{{chat=Work}}", cfg.DistributionRules[0].Filter)
	assert.Equal(t, "{{all}}", cfg.DistributionRules[1].Filter)
}
