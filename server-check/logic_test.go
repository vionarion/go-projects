package main

import (
	"strings"
	"testing"
)

func TestParseDfPercent(t *testing.T) {
	out := "Filesystem     1024-blocks     Used Available Capacity Mounted on\n" +
		"/dev/sda1         20000000 16000000   4000000      80% /\n"
	pct, err := parseDfPercent(out)
	if err != nil || pct != 80 {
		t.Fatalf("got pct=%d err=%v, want 80", pct, err)
	}
}

func TestParseMemAvailableMB(t *testing.T) {
	meminfo := "MemTotal:        2048000 kB\nMemAvailable:     512000 kB\nSwapTotal:        1000 kB\n"
	mb, err := parseMemAvailableMB(meminfo)
	if err != nil || mb != 500 {
		t.Fatalf("got mb=%d err=%v, want 500", mb, err)
	}
}

func TestSwapPercent(t *testing.T) {
	meminfo := "SwapTotal:       1000000 kB\nSwapFree:         200000 kB\n"
	pct, err := swapPercent(meminfo)
	if err != nil || pct != 80 {
		t.Fatalf("got pct=%d err=%v, want 80", pct, err)
	}

	noSwap := "SwapTotal:             0 kB\nSwapFree:              0 kB\n"
	pct, err = swapPercent(noSwap)
	if err != nil || pct != 0 {
		t.Fatalf("no swap should give 0, got pct=%d err=%v", pct, err)
	}
}

func TestPswpin(t *testing.T) {
	vmstat := "pgfault 123\npswpin 456\npswpout 789\n"
	v, err := pswpin(vmstat)
	if err != nil || v != 456 {
		t.Fatalf("got v=%d err=%v, want 456", v, err)
	}
}

func TestLoadOverThreshold(t *testing.T) {
	over, load15, err := loadOverThreshold("0.10 5.00 0.05 1/200 1234", 2, 2)
	if err != nil || !over || load15 != 5.00 {
		t.Fatalf("got over=%v load15=%v err=%v, want over=true load15=5.00", over, load15, err)
	}

	over, _, err = loadOverThreshold("0.10 3.00 0.05 1/200 1234", 2, 2)
	if err != nil || over {
		t.Fatalf("3.00 at cpus=2,maxPerCPU=2 (threshold 4) must not be over")
	}
}

func TestDecideNoProblemsClearsState(t *testing.T) {
	prev := state{sum: "x", time: 1000}
	msg, alert, st := decide(&prev, true, "", 2000, 6)
	if msg != "" || alert || st != nil {
		t.Fatalf("expected silence and cleared state, got msg=%q alert=%v st=%v", msg, alert, st)
	}
}

func TestDecideNewProblemAlerts(t *testing.T) {
	msg, alert, st := decide(nil, false, "диск полон\n", 1000, 6)
	if !alert || st == nil || st.time != 1000 {
		t.Fatalf("new problem must alert and record state, got alert=%v st=%+v", alert, st)
	}
	if msg != "диск полон\n" {
		t.Fatalf("message should be the raw problem text, got %q", msg)
	}
}

func TestDecideSameProblemWithinWindowIsSilent(t *testing.T) {
	problems := "диск полон\n"
	prev := state{sum: sumOf(problems), time: 1000}
	now := int64(1000 + 3*3600) // 3h later, window is 6h
	msg, alert, st := decide(&prev, true, problems, now, 6)
	if alert {
		t.Fatalf("same problem inside the repeat window must stay silent")
	}
	if st == nil || st.time != 1000 {
		t.Fatalf("state must be unchanged, got %+v", st)
	}
	if !strings.Contains(msg, "уже сообщал") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideSameProblemAfterWindowAlertsAgain(t *testing.T) {
	problems := "диск полон\n"
	prev := state{sum: sumOf(problems), time: 1000}
	now := int64(1000 + 7*3600) // 7h later, window is 6h
	msg, alert, st := decide(&prev, true, problems, now, 6)
	if !alert {
		t.Fatalf("same problem after the repeat window must alert again")
	}
	if st == nil || st.time != now {
		t.Fatalf("state time must refresh, got %+v", st)
	}
	if msg != problems {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideChangedProblemAlertsImmediately(t *testing.T) {
	prev := state{sum: sumOf("A\n"), time: 1000}
	msg, alert, st := decide(&prev, true, "B\n", 1000+60, 6)
	if !alert {
		t.Fatalf("a changed problem signature must alert immediately")
	}
	if st == nil || st.time != 1000+60 {
		t.Fatalf("state time must be refreshed, got %+v", st)
	}
	_ = msg
}

func TestStateRoundTrip(t *testing.T) {
	s := state{sum: "abc123", time: 222}
	parsed, ok := parseState(s.format())
	if !ok || parsed != s {
		t.Fatalf("round trip failed: got %+v ok=%v", parsed, ok)
	}
}

func TestParseStateMalformed(t *testing.T) {
	if _, ok := parseState("garbage one two"); ok {
		t.Fatalf("malformed state must not parse")
	}
}
