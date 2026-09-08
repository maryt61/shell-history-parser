// Package shellhist parses shell history files into a common
// structured form and pretty-prints them back out.
//
// Bash, zsh and fish each write history to disk in their own format,
// and the formats are easy to mis-parse: zsh embeds a start time and
// duration ahead of every command, bash's timestamped format pairs a
// comment line with the command line that follows it, and commands
// that themselves contain a newline get split across multiple lines
// with a trailing backslash. Getting any of this wrong silently
// merges or drops history entries.
package shellhist

import "time"

// Entry is one command recorded in a shell history file.
type Entry struct {
	Command string

	// Timestamp is the zero time.Time if the source format did not
	// record when the command ran.
	Timestamp time.Time

	// Duration is zero if the source format did not record how long
	// the command took to run.
	Duration time.Duration
}
