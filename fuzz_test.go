package shellhist

import "testing"

// entriesEqual reports whether two entry slices hold the same data.
// time.Time is compared with == rather than Equal because every
// Timestamp here comes from time.Unix(...).UTC(), which never carries
// a monotonic reading, so struct equality and Equal agree.
func entriesEqual(a, b []Entry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func FuzzParseBashTimestamped(f *testing.F) {
	f.Add([]byte("#1699999999\ngit status\n"))
	f.Add([]byte("#0\nls -la\n#1700000010\necho hi\n"))
	f.Add([]byte(""))
	f.Add([]byte("#1699999999\n"))
	f.Add([]byte("git status\n"))
	f.Add([]byte("#not-a-number\ngit status\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		entries, err := ParseBashTimestamped(data)
		if err != nil {
			return
		}
		again, err := ParseBashTimestamped([]byte(FormatBashTimestamped(entries)))
		if err != nil {
			t.Fatalf("re-parsing formatted output failed: %v", err)
		}
		if !entriesEqual(entries, again) {
			t.Fatalf("round trip changed entries: got %+v, want %+v", again, entries)
		}
	})
}

func FuzzParseZshExtended(f *testing.F) {
	f.Add([]byte(": 1699999999:3;git status\n"))
	f.Add([]byte(": 1699999999:1;echo one \\\necho two\n"))
	f.Add([]byte(""))
	f.Add([]byte(": 1699999999;git status\n"))
	f.Add([]byte(": notanumber:3;git status\n"))
	f.Add([]byte("git status\n"))
	f.Add([]byte(": 1699999999:3;echo one \\\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		entries, err := ParseZshExtended(data)
		if err != nil {
			return
		}
		again, err := ParseZshExtended([]byte(FormatZshExtended(entries)))
		if err != nil {
			t.Fatalf("re-parsing formatted output failed: %v", err)
		}
		if !entriesEqual(entries, again) {
			t.Fatalf("round trip changed entries: got %+v, want %+v", again, entries)
		}
	})
}
