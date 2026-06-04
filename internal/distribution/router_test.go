package distribution

import (
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/kolsha/obsidian-telegram-sync/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchedRule_AllFilter(t *testing.T) {
	rules := []config.DistributionRule{
		{Filter: "{{all}}", NotePath: "Inbox/note.md"},
	}
	msg := &models.Message{
		ID:   1,
		Text: "hello",
		From: &models.User{ID: 1, FirstName: "Test"},
		Chat: models.Chat{ID: 1, Type: "private"},
	}

	rule := MatchedRule(rules, msg)
	require.NotNil(t, rule)
	assert.Equal(t, "Inbox/note.md", rule.NotePath)
}

func TestMatchedRule_FirstMatchWins(t *testing.T) {
	rules := []config.DistributionRule{
		{Filter: "{{user=alice}}", NotePath: "Alice/note.md"},
		{Filter: "{{all}}", NotePath: "Inbox/note.md"},
	}
	msg := &models.Message{
		ID:   1,
		Text: "hello",
		From: &models.User{ID: 1, FirstName: "Bob", Username: "bob"},
		Chat: models.Chat{ID: 1, Type: "private"},
	}

	rule := MatchedRule(rules, msg)
	require.NotNil(t, rule)
	assert.Equal(t, "Inbox/note.md", rule.NotePath)
}

func TestMatchedRule_NoMatch(t *testing.T) {
	rules := []config.DistributionRule{
		{Filter: "{{user=alice}}", NotePath: "Alice/note.md"},
	}
	msg := &models.Message{
		ID:   1,
		Text: "hello",
		From: &models.User{ID: 1, FirstName: "Bob", Username: "bob"},
		Chat: models.Chat{ID: 1, Type: "private"},
	}

	rule := MatchedRule(rules, msg)
	assert.Nil(t, rule)
}

func TestMatchedRule_InvalidFilter(t *testing.T) {
	rules := []config.DistributionRule{
		{Filter: "{{broken", NotePath: "Bad/note.md"},
		{Filter: "{{all}}", NotePath: "Fallback/note.md"},
	}
	msg := &models.Message{
		ID:   1,
		Text: "hello",
		From: &models.User{ID: 1, FirstName: "Test"},
		Chat: models.Chat{ID: 1, Type: "private"},
	}

	rule := MatchedRule(rules, msg)
	require.NotNil(t, rule)
	assert.Equal(t, "Fallback/note.md", rule.NotePath)
}
