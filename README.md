# shell-history-parser

Every shell writes its history file in its own format, and the formats
don't agree on much. Fish and unconfigured bash write one command per
line. Bash with `HISTTIMEFORMAT` set writes a `#<unix-seconds>` comment
line ahead of every command. Zsh's `EXTENDED_HISTORY` writes a start
time and elapsed run time on the same line as the command, and splits
commands that contain a literal newline across multiple lines joined
by a trailing backslash.

If you've ever tried to grep, dedupe, or migrate history between
shells you've probably hit all three quirks at once: a naive
line-by-line split merges a bash timestamp into the command that
follows it, or drops half of a multi-line zsh entry.

`shellhist` parses each of these formats into a small validated
`Entry` type, and can pretty-print or reserialize the result. Parsing
returns an error with a line number for anything that doesn't match
the format, rather than silently guessing.

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/maryt61/shell-history-parser"
)

func main() {
	data, err := os.ReadFile(os.Getenv("HOME") + "/.zsh_history")
	if err != nil {
		panic(err)
	}

	entries, err := shellhist.ParseZshExtended(data)
	if err != nil {
		panic(err)
	}

	fmt.Print(shellhist.Format(entries))
}
```

`Format` prints a table like:

```
2023-11-14 09:33:19       3  git status
2023-11-14 09:33:41       0  echo hi
```

Every parser and printer in this package is a pure function: it takes
a `[]byte` or `[]Entry` in, returns a value out, and touches no global
state or file handles, which makes them straightforward to test with
plain table-driven tests.

## Supported formats

| Function                | Format                                    |
|--------------------------|--------------------------------------------|
| `ParsePlain`             | one command per line, no metadata          |
| `ParseBashTimestamped`   | bash with `HISTTIMEFORMAT` set             |
| `ParseZshExtended`       | zsh with `EXTENDED_HISTORY`                |

`FormatPlain` and `FormatZshExtended` are the corresponding inverse
operations; `Format` is a read-only pretty printer, not tied to any
one source format.

## Status

Early skeleton. Not yet handled: fish's history format, bash's
`HISTTIMEFORMAT` strftime layouts other than raw epoch seconds, and
history files that mix formats within one file.
