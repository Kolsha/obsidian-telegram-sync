package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReplacementDirectives_Literal(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTpl   string
		wantRules []replacementRule
	}{
		{
			name:      "with replacement",
			input:     "hello {{replace:world=>Go}} test",
			wantTpl:   "hello  test",
			wantRules: []replacementRule{{from: "world", to: "Go"}},
		},
		{
			name:      "delete only",
			input:     "hello {{replace:world}} test",
			wantTpl:   "hello  test",
			wantRules: []replacementRule{{from: "world", to: ""}},
		},
		{
			name:    "multiple rules preserved in order",
			input:   "{{replace:foo=>bar}}{{replace:baz}}text",
			wantTpl: "text",
			wantRules: []replacementRule{
				{from: "foo", to: "bar"},
				{from: "baz", to: ""},
			},
		},
		{
			name:      "no directives",
			input:     "plain text",
			wantTpl:   "plain text",
			wantRules: nil,
		},
		{
			name:      "empty input",
			input:     "",
			wantTpl:   "",
			wantRules: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTpl, gotRules := parseReplacementDirectives(tt.input)
			assert.Equal(t, tt.wantTpl, gotTpl)
			assert.Equal(t, tt.wantRules, gotRules)
		})
	}
}

func TestParseReplacementDirectives_Regex(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTpl   string
		wantRules []replacementRule
	}{
		{
			name:    "with replacement",
			input:   "{{replaceRe:foo.*bar=>baz}}text",
			wantTpl: "text",
			wantRules: []replacementRule{
				{from: "foo.*bar", to: "baz", isRegex: true},
			},
		},
		{
			name:    "delete only",
			input:   `{{replaceRe:\s+}}text`,
			wantTpl: "text",
			wantRules: []replacementRule{
				{from: `\s+`, to: "", isRegex: true},
			},
		},
		{
			name:    "with capture group",
			input:   `{{replaceRe:(\w+) (\w+)=>$2 $1}}text`,
			wantTpl: "text",
			wantRules: []replacementRule{
				{from: `(\w+) (\w+)`, to: "$2 $1", isRegex: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTpl, gotRules := parseReplacementDirectives(tt.input)
			assert.Equal(t, tt.wantTpl, gotTpl)
			require.Equal(t, tt.wantRules, gotRules)
		})
	}
}

func TestParseReplacementDirectives_Mixed(t *testing.T) {
	input := "{{replaceRe:foo.*=>x}}{{replace:bar=>baz}}text"
	gotTpl, gotRules := parseReplacementDirectives(input)
	assert.Equal(t, "text", gotTpl)
	require.Len(t, gotRules, 2)
	assert.Equal(t, replacementRule{from: "foo.*", to: "x", isRegex: true}, gotRules[0])
	assert.Equal(t, replacementRule{from: "bar", to: "baz", isRegex: false}, gotRules[1])
}

func TestApplyReplacementRules_Literal(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		rules []replacementRule
		want  string
	}{
		{
			name:  "simple replace",
			text:  "Hello world!",
			rules: []replacementRule{{from: "world", to: "Go"}},
			want:  "Hello Go!",
		},
		{
			name:  "delete",
			text:  "Hello world!",
			rules: []replacementRule{{from: " world", to: ""}},
			want:  "Hello!",
		},
		{
			name:  "regex special chars are treated literally",
			text:  "test.*test",
			rules: []replacementRule{{from: "test.*test", to: "fixed"}},
			want:  "fixed",
		},
		{
			name:  "newline in from",
			text:  "foo\nbar",
			rules: []replacementRule{{from: `foo\nbar`, to: "baz"}},
			want:  "baz",
		},
		{
			name:  "newline in to",
			text:  "foo",
			rules: []replacementRule{{from: "foo", to: `a\nb`}},
			want:  "a\nb",
		},
		{
			name: "rules applied in order",
			text: "abc",
			rules: []replacementRule{
				{from: "a", to: "x"},
				{from: "b", to: "y"},
			},
			want: "xyc",
		},
		{
			name:  "no rules",
			text:  "unchanged",
			rules: nil,
			want:  "unchanged",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyReplacementRules(tt.text, tt.rules)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyReplacementRules_Regex(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		rules []replacementRule
		want  string
	}{
		{
			name:  "wildcard match",
			text:  "testANYTHINGtest",
			rules: []replacementRule{{from: "test.*test", to: "fixed", isRegex: true}},
			want:  "fixed",
		},
		{
			name:  "capture group substitution",
			text:  "Hello World",
			rules: []replacementRule{{from: `Hello (\w+)`, to: "Hi $1", isRegex: true}},
			want:  "Hi World",
		},
		{
			name:  "collapse whitespace",
			text:  "  hello   world  ",
			rules: []replacementRule{{from: `\s+`, to: " ", isRegex: true}},
			want:  " hello world ",
		},
		{
			name:  "newline in replacement",
			text:  "foobar",
			rules: []replacementRule{{from: "foobar", to: `a\nb`, isRegex: true}},
			want:  "a\nb",
		},
		{
			name:  "invalid pattern silently skipped",
			text:  "original",
			rules: []replacementRule{{from: "[invalid", to: "x", isRegex: true}},
			want:  "original",
		},
		{
			name:  "invalid pattern does not block subsequent valid rules",
			text:  "hello world",
			rules: []replacementRule{
				{from: "[bad", to: "x", isRegex: true},
				{from: "world", to: "Go", isRegex: true},
			},
			want: "hello Go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyReplacementRules(tt.text, tt.rules)
			assert.Equal(t, tt.want, got)
		})
	}
}
