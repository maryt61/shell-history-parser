package shellhist

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParsePlain parses a history file that holds one command per line
// with no metadata, the format used by fish and by bash when
// HISTTIMEFORMAT is unset. Blank lines are skipped. Every returned
// Entry has a zero Timestamp and Duration since the format carries
// neither.
func ParsePlain(data []byte) ([]Entry, error) {
	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		entries = append(entries, Entry{Command: line})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("shellhist: reading plain history: %w", err)
	}
	return entries, nil
}

// ParseBashTimestamped parses bash's timestamped history format,
// produced when HISTTIMEFORMAT is set. Each entry is a "#<seconds>"
// line immediately followed by the command line it timestamps:
//
//	#1699999999
//	git status
func ParseBashTimestamped(data []byte) ([]Entry, error) {
	lines := splitLines(data)
	var entries []Entry
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			return nil, fmt.Errorf("shellhist: line %d: expected a timestamp comment, got %q", i+1, line)
		}
		sec, err := strconv.ParseInt(strings.TrimPrefix(line, "#"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("shellhist: line %d: invalid timestamp %q: %w", i+1, line, err)
		}
		i++
		if i >= len(lines) {
			return nil, fmt.Errorf("shellhist: line %d: timestamp with no command after it", i)
		}
		entries = append(entries, Entry{
			Command:   lines[i],
			Timestamp: time.Unix(sec, 0).UTC(),
		})
	}
	return entries, nil
}

// ParseZshExtended parses zsh's EXTENDED_HISTORY format:
//
//	: 1699999999:3;git status
//
// The two numbers are the command's start time in Unix seconds and
// its elapsed run time in seconds. A command that contains a literal
// newline is written across multiple lines joined by a trailing
// backslash; ParseZshExtended reassembles those into a single
// Entry.Command.
func ParseZshExtended(data []byte) ([]Entry, error) {
	lines := splitLines(data)
	var entries []Entry
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, ": ") {
			return nil, fmt.Errorf("shellhist: line %d: not a zsh extended history entry: %q", i+1, line)
		}
		rest := line[len(": "):]

		colon := strings.IndexByte(rest, ':')
		if colon < 0 {
			return nil, fmt.Errorf("shellhist: line %d: missing ':' after start time", i+1)
		}
		start, err := strconv.ParseInt(rest[:colon], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("shellhist: line %d: invalid start time %q: %w", i+1, rest[:colon], err)
		}
		rest = rest[colon+1:]

		semi := strings.IndexByte(rest, ';')
		if semi < 0 {
			return nil, fmt.Errorf("shellhist: line %d: missing ';' after elapsed time", i+1)
		}
		elapsed, err := strconv.ParseInt(rest[:semi], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("shellhist: line %d: invalid elapsed time %q: %w", i+1, rest[:semi], err)
		}

		command := rest[semi+1:]
		for strings.HasSuffix(command, "\\") && i+1 < len(lines) {
			i++
			command = command[:len(command)-1] + "\n" + lines[i]
		}

		entries = append(entries, Entry{
			Command:   command,
			Timestamp: time.Unix(start, 0).UTC(),
			Duration:  time.Duration(elapsed) * time.Second,
		})
	}
	return entries, nil
}

// ParseFishHistory parses fish's history file format, a sequence of
// YAML-like entries:
//
//	- cmd: git status
//	  when: 1699999999
//	- cmd: git commit -m "fix bug"
//	  when: 1700000010
//	  paths:
//	    - src/main.go
//
// fish does not use a real YAML parser to write this format: a "cmd"
// value is always exactly one line, with literal backslashes and
// newlines backslash-escaped as "\\" and "\n". An optional "paths"
// list, which fish uses for its own autocomplete bookkeeping, may
// follow the "when" line; ParseFishHistory validates and skips it,
// since Entry has nowhere to keep it.
func ParseFishHistory(data []byte) ([]Entry, error) {
	lines := splitLines(data)
	var entries []Entry
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "- cmd: ") {
			return nil, fmt.Errorf("shellhist: line %d: expected a '- cmd:' entry, got %q", i+1, line)
		}
		command := unescapeFishCommand(strings.TrimPrefix(line, "- cmd: "))

		i++
		if i >= len(lines) {
			return nil, fmt.Errorf("shellhist: line %d: entry has no 'when:' line", i)
		}
		whenLine := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(whenLine, "when: ") {
			return nil, fmt.Errorf("shellhist: line %d: expected a 'when:' line, got %q", i+1, lines[i])
		}
		when, err := strconv.ParseInt(strings.TrimPrefix(whenLine, "when: "), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("shellhist: line %d: invalid 'when:' timestamp %q: %w", i+1, strings.TrimPrefix(whenLine, "when: "), err)
		}

		if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "paths:" {
			i++
			for i+1 < len(lines) && strings.HasPrefix(lines[i+1], "    - ") {
				i++
			}
		}

		entries = append(entries, Entry{
			Command:   command,
			Timestamp: time.Unix(when, 0).UTC(),
		})
	}
	return entries, nil
}

// unescapeFishCommand reverses the backslash-escaping fish applies to
// a "cmd" value: "\\" becomes a literal backslash and "\n" becomes a
// newline. Any other backslash sequence is left untouched.
func unescapeFishCommand(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// splitLines splits data on "\n" without the trailing empty element
// a raw strings.Split leaves behind when data ends in a newline, so
// callers don't mistake it for a blank final entry.
func splitLines(data []byte) []string {
	s := strings.TrimSuffix(string(data), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
