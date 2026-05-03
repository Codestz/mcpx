package render

import "testing"

func TestFormatNumber(t *testing.T) {
	cases := map[int64]string{
		0:           "0",
		42:          "42",
		999:         "999",
		1234:        "1.2K",
		487231:      "487K",
		4_800_000:   "4.8M",
		120_000_000: "120M",
		1_500_000_000: "1.5B",
	}
	for in, want := range cases {
		if got := FormatNumber(in); got != want {
			t.Errorf("FormatNumber(%d): got %s want %s", in, got, want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	if FormatPercent(0.5) != "50%" {
		t.Error("0.5 → 50%")
	}
	if FormatPercent(1) != "100%" {
		t.Error("1 → 100%")
	}
	if FormatPercent(0) != "0%" {
		t.Error("0 → 0%")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[int64]string{
		412:    "412ms",
		1500:   "1s",
		59000:  "59s",
		60000:  "1m00s",
		61500:  "1m01s",
		125000: "2m05s",
	}
	for in, want := range cases {
		if got := FormatDuration(in); got != want {
			t.Errorf("FormatDuration(%d): got %s want %s", in, got, want)
		}
	}
}

func TestBar(t *testing.T) {
	if got := Bar(0, 10); got != "          " {
		t.Errorf("0%%: got %q", got)
	}
	if got := Bar(1, 10); got != "██████████" {
		t.Errorf("100%%: got %q", got)
	}
	if got := Bar(0.5, 10); got != "█████     " {
		t.Errorf("50%%: got %q", got)
	}
}

func TestSparkline(t *testing.T) {
	if got := Sparkline(nil); got != "" {
		t.Errorf("empty: got %q", got)
	}
	if got := Sparkline([]int64{0, 0, 0}); got != "▁▁▁" {
		t.Errorf("all zero: got %q", got)
	}
	got := Sparkline([]int64{1, 2, 4, 8})
	if len([]rune(got)) != 4 {
		t.Errorf("4 values → %d runes", len([]rune(got)))
	}
}

func TestTruncate(t *testing.T) {
	if Truncate("hello", 10) != "hello" {
		t.Error("under width")
	}
	if Truncate("hello world", 5) != "hell…" {
		t.Errorf("got %q", Truncate("hello world", 5))
	}
	if Truncate("", 5) != "" {
		t.Error("empty")
	}
}

func TestPadRight(t *testing.T) {
	if PadRight("hi", 5) != "hi   " {
		t.Errorf("got %q", PadRight("hi", 5))
	}
	if PadRight("hello", 3) != "hello" {
		t.Error("over width preserved")
	}
}
