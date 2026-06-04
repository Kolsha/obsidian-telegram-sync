package distribution

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-telegram/bot/models"
)

type ConditionType string

const (
	ConditionAll         ConditionType = "all"
	ConditionUser        ConditionType = "user"
	ConditionChat        ConditionType = "chat"
	ConditionTopic       ConditionType = "topic"
	ConditionForwardFrom ConditionType = "forwardFrom"
	ConditionContent     ConditionType = "content"
)

type Operation string

const (
	OpEqual      Operation = "="
	OpNotEqual   Operation = "!="
	OpContain    Operation = "~"
	OpNotContain Operation = "!~"
)

type Condition struct {
	Type      ConditionType
	Operation Operation
	Value     string
}

var conditionRegex = regexp.MustCompile(`\{\{(\w+)(!?[=~])([^}]*)\}\}`)
var allRegex = regexp.MustCompile(`\{\{all\}\}`)

func ParseFilterQuery(query string) ([]Condition, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty filter query")
	}

	if allRegex.MatchString(query) {
		return []Condition{{Type: ConditionAll}}, nil
	}

	matches := conditionRegex.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no valid conditions found in query: %s", query)
	}

	conditions := make([]Condition, 0, len(matches))
	for _, m := range matches {
		condType := ConditionType(m[1])
		switch condType {
		case ConditionUser, ConditionChat, ConditionTopic, ConditionForwardFrom, ConditionContent:
		default:
			return nil, fmt.Errorf("unknown condition type: %s", m[1])
		}

		op := Operation(m[2])
		switch op {
		case OpEqual, OpNotEqual, OpContain, OpNotContain:
		default:
			return nil, fmt.Errorf("unknown operation: %s", m[2])
		}

		conditions = append(conditions, Condition{
			Type:      condType,
			Operation: op,
			Value:     m[3],
		})
	}

	return conditions, nil
}

func EvaluateCondition(cond Condition, msg *models.Message) bool {
	if cond.Type == ConditionAll {
		return true
	}

	switch cond.Type {
	case ConditionUser:
		return evaluateUser(cond, msg)
	case ConditionChat:
		return evaluateChat(cond, msg)
	case ConditionTopic:
		return evaluateTopic(cond, msg)
	case ConditionForwardFrom:
		return evaluateForwardFrom(cond, msg)
	case ConditionContent:
		return evaluateContent(cond, msg)
	}
	return false
}

func MatchesFilter(conditions []Condition, msg *models.Message) bool {
	for _, c := range conditions {
		if !EvaluateCondition(c, msg) {
			return false
		}
	}
	return true
}

func evaluateUser(cond Condition, msg *models.Message) bool {
	if msg.From == nil {
		return cond.Operation == OpNotEqual || cond.Operation == OpNotContain
	}

	candidates := []string{
		msg.From.Username,
		strconv.FormatInt(msg.From.ID, 10),
	}
	fullName := strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
	if fullName != "" {
		candidates = append(candidates, fullName)
	}

	return matchAny(cond.Operation, cond.Value, candidates)
}

func evaluateChat(cond Condition, msg *models.Message) bool {
	chatIDStr := strconv.FormatInt(msg.Chat.ID, 10)
	chatIDStr = strings.TrimPrefix(chatIDStr, "-100")

	candidates := []string{chatIDStr}
	if msg.Chat.Title != "" {
		candidates = append(candidates, msg.Chat.Title)
	}
	if msg.Chat.Username != "" {
		candidates = append(candidates, msg.Chat.Username)
	}

	return matchAny(cond.Operation, cond.Value, candidates)
}

func evaluateTopic(cond Condition, msg *models.Message) bool {
	if !msg.IsTopicMessage {
		return cond.Operation == OpNotEqual || cond.Operation == OpNotContain
	}
	topicID := strconv.Itoa(msg.MessageThreadID)
	return matchValue(cond.Operation, cond.Value, topicID)
}

func evaluateForwardFrom(cond Condition, msg *models.Message) bool {
	name := getForwardFromName(msg)
	if name == "" {
		return cond.Operation == OpNotEqual || cond.Operation == OpNotContain
	}
	return matchValue(cond.Operation, cond.Value, name)
}

func evaluateContent(cond Condition, msg *models.Message) bool {
	candidates := []string{}
	if msg.Text != "" {
		candidates = append(candidates, msg.Text)
	}
	if msg.Caption != "" {
		candidates = append(candidates, msg.Caption)
	}
	if len(candidates) == 0 {
		return cond.Operation == OpNotEqual || cond.Operation == OpNotContain
	}
	return matchAny(cond.Operation, cond.Value, candidates)
}

func matchAny(op Operation, value string, candidates []string) bool {
	lower := strings.ToLower(value)
	switch op {
	case OpEqual:
		for _, c := range candidates {
			if strings.EqualFold(c, lower) {
				return true
			}
		}
		return false
	case OpNotEqual:
		for _, c := range candidates {
			if strings.EqualFold(c, lower) {
				return false
			}
		}
		return true
	case OpContain:
		for _, c := range candidates {
			if strings.Contains(strings.ToLower(c), lower) {
				return true
			}
		}
		return false
	case OpNotContain:
		for _, c := range candidates {
			if strings.Contains(strings.ToLower(c), lower) {
				return false
			}
		}
		return true
	}
	return false
}

func matchValue(op Operation, expected, actual string) bool {
	lower := strings.ToLower(expected)
	actualLower := strings.ToLower(actual)
	switch op {
	case OpEqual:
		return strings.EqualFold(actualLower, lower)
	case OpNotEqual:
		return !strings.EqualFold(actualLower, lower)
	case OpContain:
		return strings.Contains(actualLower, lower)
	case OpNotContain:
		return !strings.Contains(actualLower, lower)
	}
	return false
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
		if c.SenderChat.Title != "" {
			return c.SenderChat.Title
		}
		return c.SenderChat.Username
	case models.MessageOriginTypeChannel:
		c := msg.ForwardOrigin.MessageOriginChannel
		if c == nil {
			return ""
		}
		if c.Chat.Title != "" {
			return c.Chat.Title
		}
		return c.Chat.Username
	}
	return ""
}
