package distribution

import (
	"github.com/go-telegram/bot/models"
	"github.com/kolsha/obsidian-telegram-sync/internal/config"
)

// MatchedRule finds the first distribution rule that matches the message.
// Returns nil if no rule matches.
func MatchedRule(rules []config.DistributionRule, msg *models.Message) *config.DistributionRule {
	for i := range rules {
		conditions, err := ParseFilterQuery(rules[i].Filter)
		if err != nil {
			continue
		}
		if MatchesFilter(conditions, msg) {
			return &rules[i]
		}
	}
	return nil
}
