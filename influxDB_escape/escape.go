package influxDB_escape

import (
	"regexp"
	"strings"
)

var (
	escapeKeyRegex   = regexp.MustCompile(`[\n ,=\\]`)
	escapeValueRegex = regexp.MustCompile(`["\\]`)
)

func EscapeKey(key string) string {
	return escapeKeyRegex.ReplaceAllString(key, `\$1`)
}

func EscapeValue(value string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	sb.WriteString(escapeValueRegex.ReplaceAllString(value, `\$1`))
	sb.WriteByte('"')
	return sb.String()
}
