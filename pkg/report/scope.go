package report

import (
	"fmt"
	"regexp"
	"strings"
)

var projectKey = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// Scope builds a project-bounded JQL expression. A query is a filter expression,
// not a complete JQL query: it must not escape the surrounding parentheses.
func Scope(project, query string) (string, error) {
	if !projectKey.MatchString(project) {
		return "", fmt.Errorf("report requires a Jira project key from --jira-project or --profile-name (letters, digits, underscores)")
	}
	query = strings.TrimSpace(query)
	jql := fmt.Sprintf(`project = "%s"`, project)
	if query == "" {
		return jql, nil
	}
	depth := 0
	var quote byte
	var unquoted strings.Builder
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				if i >= len(query) {
					return "", fmt.Errorf("query ends in an incomplete escape")
				}
			} else if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '\'', '"':
			quote = ch
			unquoted.WriteByte(' ')
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", fmt.Errorf("query has unbalanced parentheses")
			}
		case ';', '\\':
			return "", fmt.Errorf("query contains unsupported syntax")
		default:
			unquoted.WriteByte(ch)
		}
	}
	if quote != 0 || depth != 0 {
		return "", fmt.Errorf("query has unbalanced quotes or parentheses")
	}
	plain := strings.ToLower(unquoted.String())
	words := strings.Fields(plain)
	for _, word := range words {
		if word == "order" {
			return "", fmt.Errorf("query must be a filter expression without ORDER BY")
		}
	}
	for _, comment := range []string{"--", "//", "/*", "*/"} {
		if strings.Contains(plain, comment) {
			return "", fmt.Errorf("query contains unsupported comment syntax")
		}
	}
	return jql + " AND (" + query + ")", nil
}
