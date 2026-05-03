package detect

import (
	"regexp"
	"strings"
)

var (
	uuidPattern = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	ipv4Pattern = regexp.MustCompile(`^\d{1,3}(?:\.\d{1,3}){3}$`)
	hexPattern  = regexp.MustCompile(`^(?:0x)?[a-f0-9]{8,}$`)
	numPattern  = regexp.MustCompile(`^[-+]?\d+(?:\.\d+)?$`)
)

func NormalizeMessage(message string) string {
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(message)))
	if len(tokens) == 0 {
		return "<empty>"
	}

	normalized := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.Trim(token, "[](){}<>,;\"'")
		token = normalizeToken(token)
		if token == "" {
			continue
		}
		normalized = append(normalized, token)
	}

	if len(normalized) == 0 {
		return "<empty>"
	}

	return strings.Join(normalized, " ")
}

func normalizeToken(token string) string {
	if token == "" {
		return ""
	}
	if key, value, ok := strings.Cut(token, "="); ok && key != "" && value != "" {
		return key + "=" + normalizeAtomicToken(value)
	}
	return normalizeAtomicToken(token)
}

func normalizeAtomicToken(token string) string {
	switch {
	case uuidPattern.MatchString(token):
		return "<uuid>"
	case ipv4Pattern.MatchString(token):
		return "<ip>"
	case hexPattern.MatchString(token):
		return "<hex>"
	case numPattern.MatchString(token):
		return "<num>"
	case strings.Count(token, "/") >= 2:
		return "<path>"
	default:
		return token
	}
}
