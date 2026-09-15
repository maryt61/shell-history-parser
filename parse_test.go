package shellhist

import (
	"testing"
	"time"
)

func TestParsePlain(t *testing.T) {
	in := "git status\n\nls -la\n"
	entries, err := ParsePlain([]byte(in))
	if err != nil {
		t.Fatalf("ParsePlain: %v", err)
	}
	want := []Entry{{Command: "git status"}, {Command: "ls -la"}}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}
	for i, e := range entries {
		if e.Command != want[i].Command || !e.Timestamp.IsZero() || e.Duration != 0 {
			t.Errorf("entry %d = %+v, want %+v", i, e, want[i])
		}
	}
}

func TestParseBashTimestamped(t *testing.T) {
	in := "#1699999999\ngit status\n#1700000010\nls -la\n"
	entries, err := ParseBashTimestamped([]byte(in))
	if err != nil {
		t.Fatalf("ParseBashTimestamped: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Command != "git status" || entries[0].Timestamp.Unix() != 1699999999 {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].Command != "ls -la" || entries[1].Timestamp.Unix() != 1700000010 {
		t.Errorf("entry 1 = %+v", entries[1])
	}
}

func TestParseBashTimestampedRejectsMissingCommand(t *testing.T) {
	_, err := ParseBashTimestamped([]byte("#1699999999\n"))
	if err == nil {
		t.Fatal("expected an error for a timestamp with no command")
	}
}

func TestParseBashTimestampedRejectsBadLine(t *testing.T) {
	_, err := ParseBashTimestamped([]byte("git status\n"))
	if err == nil {
		t.Fatal("expected an error for a line that isn't a timestamp comment")
	}
}

func TestFormatBashTimestampedRoundTrip(t *testing.T) {
	in := "#1699999999\ngit status\n#1700000010\nls -la\n"
	entries, err := ParseBashTimestamped([]byte(in))
	if err != nil {
		t.Fatalf("ParseBashTimestamped: %v", err)
	}
	if out := FormatBashTimestamped(entries); out != in {
		t.Errorf("round trip mismatch:\ngot:  %q\nwant: %q", out, in)
	}
}

func TestParseZshExtended(t *testing.T) {
	in := ": 1699999999:3;git status\n: 1700000010:0;echo hi\n"
	entries, err := ParseZshExtended([]byte(in))
	if err != nil {
		t.Fatalf("ParseZshExtended: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Command != "git status" || entries[0].Duration != 3*time.Second {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].Command != "echo hi" || entries[1].Duration != 0 {
		t.Errorf("entry 1 = %+v", entries[1])
	}
}

func TestParseZshExtendedMultilineCommand(t *testing.T) {
	in := ": 1699999999:1;echo one \\\necho two\n"
	entries, err := ParseZshExtended([]byte(in))
	if err != nil {
		t.Fatalf("ParseZshExtended: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	want := "echo one \necho two"
	if entries[0].Command != want {
		t.Errorf("command = %q, want %q", entries[0].Command, want)
	}
}

func TestParseZshExtendedRejectsBadLine(t *testing.T) {
	_, err := ParseZshExtended([]byte("git status\n"))
	if err == nil {
		t.Fatal("expected an error for a line missing the ': start:elapsed;' prefix")
	}
}

func TestFormatZshExtendedRoundTrip(t *testing.T) {
	in := ": 1699999999:3;git status\n: 1700000010:0;echo one \\\necho two\n"
	entries, err := ParseZshExtended([]byte(in))
	if err != nil {
		t.Fatalf("ParseZshExtended: %v", err)
	}
	out := FormatZshExtended(entries)
	if out != in {
		t.Errorf("round trip mismatch:\ngot:  %q\nwant: %q", out, in)
	}
}

func TestParseFishHistory(t *testing.T) {
	in := "- cmd: git status\n  when: 1699999999\n- cmd: git commit -m \"fix bug\"\n  when: 1700000010\n  paths:\n    - src/main.go\n    - src/other.go\n"
	entries, err := ParseFishHistory([]byte(in))
	if err != nil {
		t.Fatalf("ParseFishHistory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Command != "git status" || entries[0].Timestamp.Unix() != 1699999999 {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].Command != `git commit -m "fix bug"` || entries[1].Timestamp.Unix() != 1700000010 {
		t.Errorf("entry 1 = %+v", entries[1])
	}
}

func TestParseFishHistoryEscapedCommand(t *testing.T) {
	in := "- cmd: echo one \\n echo two \\\\ done\n  when: 1699999999\n"
	entries, err := ParseFishHistory([]byte(in))
	if err != nil {
		t.Fatalf("ParseFishHistory: %v", err)
	}
	want := "echo one \n echo two \\ done"
	if len(entries) != 1 || entries[0].Command != want {
		t.Fatalf("got %+v, want command %q", entries, want)
	}
}

func TestParseFishHistoryRejectsBadEntry(t *testing.T) {
	_, err := ParseFishHistory([]byte("cmd: git status\nwhen: 1699999999\n"))
	if err == nil {
		t.Fatal("expected an error for a line missing the '- cmd:' prefix")
	}
}

func TestParseFishHistoryRejectsMissingWhen(t *testing.T) {
	_, err := ParseFishHistory([]byte("- cmd: git status\n"))
	if err == nil {
		t.Fatal("expected an error for an entry with no 'when:' line")
	}
}

func TestParseFishHistoryRejectsBadWhen(t *testing.T) {
	_, err := ParseFishHistory([]byte("- cmd: git status\n  when: not-a-number\n"))
	if err == nil {
		t.Fatal("expected an error for an invalid 'when:' timestamp")
	}
}

func TestFormatFishHistoryRoundTrip(t *testing.T) {
	in := "- cmd: git status\n  when: 1699999999\n- cmd: echo one \\n two\n  when: 1700000010\n"
	entries, err := ParseFishHistory([]byte(in))
	if err != nil {
		t.Fatalf("ParseFishHistory: %v", err)
	}
	if out := FormatFishHistory(entries); out != in {
		t.Errorf("round trip mismatch:\ngot:  %q\nwant: %q", out, in)
	}
}

func TestFormatPlainRoundTrip(t *testing.T) {
	in := "git status\nls -la\n"
	entries, err := ParsePlain([]byte(in))
	if err != nil {
		t.Fatalf("ParsePlain: %v", err)
	}
	if out := FormatPlain(entries); out != in {
		t.Errorf("round trip mismatch: got %q, want %q", out, in)
	}
}
