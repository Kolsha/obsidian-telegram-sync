package markdown

import (
	"regexp"
	"strings"
)

var (
	replaceReWithRE = regexp.MustCompile(`\{\{replaceRe:(.*?)=>(.*?)\}\}`)
	replaceReOnlyRE = regexp.MustCompile(`\{\{replaceRe:(.*?)\}\}`)
	replaceWithRE   = regexp.MustCompile(`\{\{replace:(.*?)=>(.*?)\}\}`)
	replaceOnlyRE   = regexp.MustCompile(`\{\{replace:(.*?)\}\}`)
)

// replacementRule represents a single text replacement operation.
type replacementRule struct {
	// from is the pattern to match.
	from string
	// to is the substitution string; supports \n for newline.
	to string
	// isRegex, when true, treats from as a Go regular expression;
	// when false, treats it as a literal string (with \n support).
	isRegex bool
}

// parseReplacementDirectives scans template for {{replace:...}} and {{replaceRe:...}}
// directives, removes them in-place, and returns the cleaned template together with
// the ordered list of replacement rules.
//
// Supported syntax:
//
//	{{replace:literal=>replacement}}   – replace literal string with replacement
//	{{replace:literal}}                – delete literal string
//	{{replaceRe:pattern=>replacement}} – replace regex match with replacement
//	{{replaceRe:pattern}}              – delete regex match
//
// In all forms, \n inside literal or replacement strings is interpreted as a newline.
// In regex forms, the full Go regexp syntax is available in pattern.
func parseReplacementDirectives(template string) (string, []replacementRule) {
	var rules []replacementRule

	makeExtractor := func(re *regexp.Regexp, isRegex bool) func(string) string {
		return func(match string) string {
			parts := re.FindStringSubmatch(match)
			to := ""
			if len(parts) > 2 {
				to = parts[2]
			}
			rules = append(rules, replacementRule{from: parts[1], to: to, isRegex: isRegex})
			return ""
		}
	}

	// Process the "with replacement" variants first to avoid the simpler pattern
	// consuming the => separator as part of the from-field.
	result := template
	result = replaceReWithRE.ReplaceAllStringFunc(result, makeExtractor(replaceReWithRE, true))
	result = replaceReOnlyRE.ReplaceAllStringFunc(result, makeExtractor(replaceReOnlyRE, true))
	result = replaceWithRE.ReplaceAllStringFunc(result, makeExtractor(replaceWithRE, false))
	result = replaceOnlyRE.ReplaceAllStringFunc(result, makeExtractor(replaceOnlyRE, false))

	return result, rules
}

// applyReplacementRules applies a list of replacement rules to text in order.
//
// For literal rules (isRegex == false) the from value is treated as a plain string;
// \n sequences are expanded to actual newlines both in from and to.
//
// For regex rules (isRegex == true) the from value is used as a Go regular expression
// and the full regexp replacement syntax ($1, $2, …) is available in to.
// Invalid patterns are silently skipped so a bad template cannot crash the process.
func applyReplacementRules(text string, rules []replacementRule) string {
	for _, rule := range rules {
		var pattern string
		if rule.isRegex {
			pattern = rule.from
		} else {
			pattern = regexp.QuoteMeta(rule.from)
			// QuoteMeta turns \n → \\n; undo that so literal \n matches newlines.
			pattern = strings.ReplaceAll(pattern, `\\n`, `\n`)
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}

		to := strings.ReplaceAll(rule.to, `\n`, "\n")
		text = re.ReplaceAllString(text, to)
	}
	return text
}
