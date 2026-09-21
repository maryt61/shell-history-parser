# shell-history-parser

Every shell writes its history file in its own format, and the formats
don't agree on much. Unconfigured bash writes one command per line.
Bash with `HISTTIMEFORMAT` set writes a `#<unix-seconds>` comment line
ahead of every command. Zsh's `EXTENDED_HISTORY` writes a start time
and elapsed run time on the same line as the command, and splits
commands that contain a literal newline across multiple lines joined
by a trailing backslash. Fish writes a YAML-like `- cmd: ... / when:
...` block per entry, with backslashes and newlines inside the command
itself backslash-escaped rather than written literally.

If you've ever tried to grep, dedupe, or migrate history between
shells you've probably hit these quirks at once: a naive line-by-line
split merges a bash timestamp into the command that follows it, drops
half of a multi-line zsh entry, or takes a fish `paths:` line for a
new command.

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

## CLI

`cmd/shellhist` is a thin wrapper over the library for one-off use
from a terminal:

```sh
go run ./cmd/shellhist cat -format=zsh ~/.zsh_history
go run ./cmd/shellhist convert -from=zsh -to=plain ~/.zsh_history
go run ./cmd/shellhist merge -format=bash ~/.bash_history laptop.bash_history
```

`cat` parses one or more files and prints the table `Format` produces.
`convert` reparses a file in one format and reprints it in another.
`merge` parses two or more files of the same format, concatenates
their entries, sorts them by timestamp (stable, so entries with no
timestamp keep the order they were read in), and prints the result
back in that format. Formats are named `plain`, `bash`, `zsh` and
`fish`, matching the parser names above.

## Supported formats

| Function                | Format                                    |
|--------------------------|--------------------------------------------|
| `ParsePlain`             | one command per line, no metadata          |
| `ParseBashTimestamped`   | bash with `HISTTIMEFORMAT` set             |
| `ParseZshExtended`       | zsh with `EXTENDED_HISTORY`                |
| `ParseFishHistory`       | fish's `- cmd: / when:` format             |

`FormatPlain`, `FormatBashTimestamped`, `FormatZshExtended` and
`FormatFishHistory` are the corresponding inverse operations; `Format`
is a read-only pretty printer, not tied to any one source format.
`FormatFishHistory` drops the `paths:` block fish attaches to some
entries, since `Entry` has no field to hold it.

## Status

Early skeleton. `HISTTIMEFORMAT` only changes how bash's `history`
builtin displays timestamps; the `~/.bash_history` file itself always
stores raw epoch seconds, so `ParseBashTimestamped` already covers it.
Not yet handled: history files that mix formats within one file, or
that have a corrupted trailing entry.
