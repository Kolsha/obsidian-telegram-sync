package distribution

import (
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilterQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    []Condition
		wantErr bool
	}{
		{
			name:  "all",
			query: "{{all}}",
			want:  []Condition{{Type: ConditionAll}},
		},
		{
			name:  "user equal",
			query: "{{user=johndoe}}",
			want:  []Condition{{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"}},
		},
		{
			name:  "user not equal",
			query: "{{user!=johndoe}}",
			want:  []Condition{{Type: ConditionUser, Operation: OpNotEqual, Value: "johndoe"}},
		},
		{
			name:  "chat contains",
			query: "{{chat~Work}}",
			want:  []Condition{{Type: ConditionChat, Operation: OpContain, Value: "Work"}},
		},
		{
			name:  "chat not contains",
			query: "{{chat!~Work}}",
			want:  []Condition{{Type: ConditionChat, Operation: OpNotContain, Value: "Work"}},
		},
		{
			name:  "topic equal",
			query: "{{topic=General}}",
			want:  []Condition{{Type: ConditionTopic, Operation: OpEqual, Value: "General"}},
		},
		{
			name:  "forwardFrom equal",
			query: "{{forwardFrom=Alice}}",
			want:  []Condition{{Type: ConditionForwardFrom, Operation: OpEqual, Value: "Alice"}},
		},
		{
			name:  "content contains",
			query: "{{content~keyword}}",
			want:  []Condition{{Type: ConditionContent, Operation: OpContain, Value: "keyword"}},
		},
		{
			name:  "content not contains",
			query: "{{content!~keyword}}",
			want:  []Condition{{Type: ConditionContent, Operation: OpNotContain, Value: "keyword"}},
		},
		{
			name:  "multiple conditions",
			query: "{{user=johndoe}}{{chat=Work}}{{content~hello}}",
			want: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
				{Type: ConditionChat, Operation: OpEqual, Value: "Work"},
				{Type: ConditionContent, Operation: OpContain, Value: "hello"},
			},
		},
		{
			name:  "multiple conditions with whitespace",
			query: "  {{user=johndoe}} {{chat=Work}}  ",
			want: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
				{Type: ConditionChat, Operation: OpEqual, Value: "Work"},
			},
		},
		{
			name:    "empty query",
			query:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only",
			query:   "   ",
			wantErr: true,
		},
		{
			name:    "invalid pattern",
			query:   "hello world",
			wantErr: true,
		},
		{
			name:    "unknown condition type",
			query:   "{{invalid=value}}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilterQuery(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func baseMsg() *models.Message {
	return &models.Message{
		From: &models.User{
			ID:        12345,
			Username:  "johndoe",
			FirstName: "John",
			LastName:  "Doe",
		},
		Chat: models.Chat{
			ID:       -1001234567890,
			Type:     "supergroup",
			Title:    "Work Chat",
			Username: "workchat",
		},
		Text: "Hello world! This is a test message.",
	}
}

func TestEvaluateCondition_All(t *testing.T) {
	msg := baseMsg()
	assert.True(t, EvaluateCondition(Condition{Type: ConditionAll}, msg))
}

func TestEvaluateCondition_User(t *testing.T) {
	msg := baseMsg()

	tests := []struct {
		name string
		cond Condition
		want bool
	}{
		{
			name: "username equal match",
			cond: Condition{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
			want: true,
		},
		{
			name: "username equal no match",
			cond: Condition{Type: ConditionUser, Operation: OpEqual, Value: "janedoe"},
			want: false,
		},
		{
			name: "user ID equal match",
			cond: Condition{Type: ConditionUser, Operation: OpEqual, Value: "12345"},
			want: true,
		},
		{
			name: "full name equal match",
			cond: Condition{Type: ConditionUser, Operation: OpEqual, Value: "John Doe"},
			want: true,
		},
		{
			name: "username not equal match",
			cond: Condition{Type: ConditionUser, Operation: OpNotEqual, Value: "janedoe"},
			want: true,
		},
		{
			name: "username not equal no match",
			cond: Condition{Type: ConditionUser, Operation: OpNotEqual, Value: "johndoe"},
			want: false,
		},
		{
			name: "username contains match",
			cond: Condition{Type: ConditionUser, Operation: OpContain, Value: "john"},
			want: true,
		},
		{
			name: "username contains no match",
			cond: Condition{Type: ConditionUser, Operation: OpContain, Value: "xyz"},
			want: false,
		},
		{
			name: "username not contains match",
			cond: Condition{Type: ConditionUser, Operation: OpNotContain, Value: "xyz"},
			want: true,
		},
		{
			name: "username not contains no match",
			cond: Condition{Type: ConditionUser, Operation: OpNotContain, Value: "john"},
			want: false,
		},
		{
			name: "case insensitive match",
			cond: Condition{Type: ConditionUser, Operation: OpEqual, Value: "JohnDoe"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateCondition(tt.cond, msg))
		})
	}
}

func TestEvaluateCondition_User_NilFrom(t *testing.T) {
	msg := baseMsg()
	msg.From = nil

	assert.False(t, EvaluateCondition(Condition{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"}, msg))
	assert.True(t, EvaluateCondition(Condition{Type: ConditionUser, Operation: OpNotEqual, Value: "johndoe"}, msg))
}

func TestEvaluateCondition_Chat(t *testing.T) {
	msg := baseMsg()

	tests := []struct {
		name string
		cond Condition
		want bool
	}{
		{
			name: "chat title equal match",
			cond: Condition{Type: ConditionChat, Operation: OpEqual, Value: "Work Chat"},
			want: true,
		},
		{
			name: "chat username equal match",
			cond: Condition{Type: ConditionChat, Operation: OpEqual, Value: "workchat"},
			want: true,
		},
		{
			name: "chat ID equal match (stripped -100)",
			cond: Condition{Type: ConditionChat, Operation: OpEqual, Value: "1234567890"},
			want: true,
		},
		{
			name: "chat title contains",
			cond: Condition{Type: ConditionChat, Operation: OpContain, Value: "Work"},
			want: true,
		},
		{
			name: "chat title not contains",
			cond: Condition{Type: ConditionChat, Operation: OpNotContain, Value: "Personal"},
			want: true,
		},
		{
			name: "chat not equal match",
			cond: Condition{Type: ConditionChat, Operation: OpNotEqual, Value: "Other Chat"},
			want: true,
		},
		{
			name: "chat not equal no match",
			cond: Condition{Type: ConditionChat, Operation: OpNotEqual, Value: "Work Chat"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateCondition(tt.cond, msg))
		})
	}
}

func TestEvaluateCondition_Topic(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		cond Condition
		want bool
	}{
		{
			name: "topic match by thread ID",
			msg: &models.Message{
				Chat:            models.Chat{ID: 1},
				IsTopicMessage:  true,
				MessageThreadID: 42,
			},
			cond: Condition{Type: ConditionTopic, Operation: OpEqual, Value: "42"},
			want: true,
		},
		{
			name: "topic no match",
			msg: &models.Message{
				Chat:            models.Chat{ID: 1},
				IsTopicMessage:  true,
				MessageThreadID: 42,
			},
			cond: Condition{Type: ConditionTopic, Operation: OpEqual, Value: "99"},
			want: false,
		},
		{
			name: "not a topic message",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
			},
			cond: Condition{Type: ConditionTopic, Operation: OpEqual, Value: "42"},
			want: false,
		},
		{
			name: "not topic - not equal returns true",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
			},
			cond: Condition{Type: ConditionTopic, Operation: OpNotEqual, Value: "42"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateCondition(tt.cond, tt.msg))
		})
	}
}

func TestEvaluateCondition_ForwardFrom(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		cond Condition
		want bool
	}{
		{
			name: "forward from user match",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeUser,
					MessageOriginUser: &models.MessageOriginUser{
						SenderUser: models.User{
							FirstName: "Alice",
							LastName:  "Smith",
						},
					},
				},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpEqual, Value: "Alice Smith"},
			want: true,
		},
		{
			name: "forward from user contains",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeUser,
					MessageOriginUser: &models.MessageOriginUser{
						SenderUser: models.User{FirstName: "Alice"},
					},
				},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpContain, Value: "Ali"},
			want: true,
		},
		{
			name: "forward from hidden user",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeHiddenUser,
					MessageOriginHiddenUser: &models.MessageOriginHiddenUser{
						SenderUserName: "HiddenAlice",
					},
				},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpEqual, Value: "HiddenAlice"},
			want: true,
		},
		{
			name: "forward from chat",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeChat,
					MessageOriginChat: &models.MessageOriginChat{
						SenderChat: models.Chat{Title: "News Channel"},
					},
				},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpEqual, Value: "News Channel"},
			want: true,
		},
		{
			name: "forward from channel",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
				ForwardOrigin: &models.MessageOrigin{
					Type: models.MessageOriginTypeChannel,
					MessageOriginChannel: &models.MessageOriginChannel{
						Chat: models.Chat{Title: "My Channel"},
					},
				},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpContain, Value: "Channel"},
			want: true,
		},
		{
			name: "no forward origin",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpEqual, Value: "Alice"},
			want: false,
		},
		{
			name: "no forward origin - not equal returns true",
			msg: &models.Message{
				Chat: models.Chat{ID: 1},
			},
			cond: Condition{Type: ConditionForwardFrom, Operation: OpNotEqual, Value: "Alice"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateCondition(tt.cond, tt.msg))
		})
	}
}

func TestEvaluateCondition_Content(t *testing.T) {
	tests := []struct {
		name string
		msg  *models.Message
		cond Condition
		want bool
	}{
		{
			name: "text contains",
			msg:  &models.Message{Chat: models.Chat{ID: 1}, Text: "Hello world"},
			cond: Condition{Type: ConditionContent, Operation: OpContain, Value: "hello"},
			want: true,
		},
		{
			name: "text equal",
			msg:  &models.Message{Chat: models.Chat{ID: 1}, Text: "Hello world"},
			cond: Condition{Type: ConditionContent, Operation: OpEqual, Value: "Hello world"},
			want: true,
		},
		{
			name: "caption contains",
			msg:  &models.Message{Chat: models.Chat{ID: 1}, Caption: "Photo caption here"},
			cond: Condition{Type: ConditionContent, Operation: OpContain, Value: "caption"},
			want: true,
		},
		{
			name: "text not contains match",
			msg:  &models.Message{Chat: models.Chat{ID: 1}, Text: "Hello world"},
			cond: Condition{Type: ConditionContent, Operation: OpNotContain, Value: "goodbye"},
			want: true,
		},
		{
			name: "text not contains no match",
			msg:  &models.Message{Chat: models.Chat{ID: 1}, Text: "Hello world"},
			cond: Condition{Type: ConditionContent, Operation: OpNotContain, Value: "hello"},
			want: false,
		},
		{
			name: "empty content - contains returns false",
			msg:  &models.Message{Chat: models.Chat{ID: 1}},
			cond: Condition{Type: ConditionContent, Operation: OpContain, Value: "anything"},
			want: false,
		},
		{
			name: "empty content - not contains returns true",
			msg:  &models.Message{Chat: models.Chat{ID: 1}},
			cond: Condition{Type: ConditionContent, Operation: OpNotContain, Value: "anything"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EvaluateCondition(tt.cond, tt.msg))
		})
	}
}

func TestMatchesFilter(t *testing.T) {
	msg := baseMsg()

	tests := []struct {
		name       string
		conditions []Condition
		want       bool
	}{
		{
			name:       "all matches",
			conditions: []Condition{{Type: ConditionAll}},
			want:       true,
		},
		{
			name:       "empty conditions matches",
			conditions: []Condition{},
			want:       true,
		},
		{
			name: "single matching condition",
			conditions: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
			},
			want: true,
		},
		{
			name: "single non-matching condition",
			conditions: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "janedoe"},
			},
			want: false,
		},
		{
			name: "multiple conditions all match",
			conditions: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
				{Type: ConditionChat, Operation: OpContain, Value: "Work"},
				{Type: ConditionContent, Operation: OpContain, Value: "Hello"},
			},
			want: true,
		},
		{
			name: "multiple conditions one fails",
			conditions: []Condition{
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
				{Type: ConditionChat, Operation: OpEqual, Value: "Personal"},
				{Type: ConditionContent, Operation: OpContain, Value: "Hello"},
			},
			want: false,
		},
		{
			name: "all with other conditions",
			conditions: []Condition{
				{Type: ConditionAll},
				{Type: ConditionUser, Operation: OpEqual, Value: "johndoe"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MatchesFilter(tt.conditions, msg))
		})
	}
}
