// Command shellhist is a thin CLI over the shellhist package: read
// one or more history files in a known format, and print them back
// out as a table, in another format, or merged together in timestamp
// order.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/maryt61/shell-history-parser"
)

var parsers = map[string]func([]byte) ([]shellhist.Entry, error){
	"plain": shellhist.ParsePlain,
	"bash":  shellhist.ParseBashTimestamped,
	"zsh":   shellhist.ParseZshExtended,
	"fish":  shellhist.ParseFishHistory,
}

var printers = map[string]func([]shellhist.Entry) string{
	"plain": shellhist.FormatPlain,
	"bash":  shellhist.FormatBashTimestamped,
	"zsh":   shellhist.FormatZshExtended,
	"fish":  shellhist.FormatFishHistory,
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "cat":
		err = runCat(os.Args[2:])
	case "convert":
		err = runConvert(os.Args[2:])
	case "merge":
		err = runMerge(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "shellhist: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "shellhist:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  shellhist cat -format=<plain|bash|zsh|fish> <file>...
  shellhist convert -from=<format> -to=<format> <file>
  shellhist merge -format=<format> <file>...

formats: plain, bash, zsh, fish`)
}

func parserFor(format string) (func([]byte) ([]shellhist.Entry, error), error) {
	p, ok := parsers[format]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (want plain, bash, zsh or fish)", format)
	}
	return p, nil
}

func printerFor(format string) (func([]shellhist.Entry) string, error) {
	p, ok := printers[format]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (want plain, bash, zsh or fish)", format)
	}
	return p, nil
}

// readAll parses every file in paths with parse and returns their
// entries concatenated in the order the files were given.
func readAll(paths []string, parse func([]byte) ([]shellhist.Entry, error)) ([]shellhist.Entry, error) {
	var entries []shellhist.Entry
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		parsed, err := parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		entries = append(entries, parsed...)
	}
	return entries, nil
}

func runCat(args []string) error {
	fs := flag.NewFlagSet("cat", flag.ExitOnError)
	format := fs.String("format", "", "history file format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *format == "" {
		return errors.New("cat: -format is required")
	}
	if fs.NArg() == 0 {
		return errors.New("cat: at least one file is required")
	}
	parse, err := parserFor(*format)
	if err != nil {
		return fmt.Errorf("cat: %w", err)
	}
	entries, err := readAll(fs.Args(), parse)
	if err != nil {
		return fmt.Errorf("cat: %w", err)
	}
	fmt.Print(shellhist.Format(entries))
	return nil
}

func runConvert(args []string) error {
	fs := flag.NewFlagSet("convert", flag.ExitOnError)
	from := fs.String("from", "", "source format")
	to := fs.String("to", "", "destination format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *to == "" {
		return errors.New("convert: -from and -to are required")
	}
	if fs.NArg() != 1 {
		return errors.New("convert: exactly one file is required")
	}
	parse, err := parserFor(*from)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	print, err := printerFor(*to)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	entries, err := readAll(fs.Args(), parse)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}
	fmt.Print(print(entries))
	return nil
}

func runMerge(args []string) error {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	format := fs.String("format", "", "history file format")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *format == "" {
		return errors.New("merge: -format is required")
	}
	if fs.NArg() < 2 {
		return errors.New("merge: at least two files are required")
	}
	parse, err := parserFor(*format)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	print, err := printerFor(*format)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	entries, err := readAll(fs.Args(), parse)
	if err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	// Stable so entries from the same source file, or with no
	// timestamp at all, keep the order they were read in.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	fmt.Print(print(entries))
	return nil
}
