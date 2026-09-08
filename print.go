package shellhist

import (
	"fmt"
	"strings"
)

// Format renders entries as a human-readable table: a timestamp
// column (or "-" if unknown), a duration column when any entry has
// one, and the command itself.
func Format(entries []Entry) string {
	showDuration := false
	for _, e := range entries {
		if e.Duration != 0 {
			showDuration = true
			break
		}
	}

	var b strings.Builder
	for _, e := range entries {
		ts := "-"
		if !e.Timestamp.IsZero() {
			ts = e.Timestamp.UTC().Format("2006-01-02 15:04:05")
		}
		if showDuration {
			fmt.Fprintf(&b, "%s  %6s  %s\n", ts, e.Duration, e.Command)
		} else {
			fmt.Fprintf(&b, "%s  %s\n", ts, e.Command)
		}
	}
	return b.String()
}

// FormatPlain renders entries as one command per line, discarding any
// timestamp or duration. It is the inverse of ParsePlain.
func FormatPlain(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.Command)
		b.WriteByte('\n')
	}
	return b.String()
}

// FormatZshExtended renders entries back into zsh's EXTENDED_HISTORY
// format, the inverse of ParseZshExtended. Commands containing a
// newline are split back across "\"-continued lines.
func FormatZshExtended(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		lines := strings.Split(e.Command, "\n")
		var start int64
		if !e.Timestamp.IsZero() {
			start = e.Timestamp.Unix()
		}
		elapsed := int64(e.Duration.Seconds())

		fmt.Fprintf(&b, ": %d:%d;%s", start, elapsed, lines[0])
		for _, cont := range lines[1:] {
			b.WriteString("\\\n")
			b.WriteString(cont)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
