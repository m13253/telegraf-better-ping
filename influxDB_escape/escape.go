package influxDB_escape

import (
	"regexp"
	"strings"
)

var (
	escapeKeyRegex   = regexp.MustCompile(`([ ,=\\])`)
	escapeValueRegex = regexp.MustCompile(`(["\\])`)
)

func EscapeKey(key string) string {
	return escapeKeyRegex.ReplaceAllString(strings.ReplaceAll(key, "\n", " "), `\$1`)
}

func EscapeValue(value string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	sb.WriteString(escapeValueRegex.ReplaceAllString(strings.ReplaceAll(value, "\n", " "), `\$1`))
	sb.WriteByte('"')
	return sb.String()
}
