package main

import (
	"strings"
	"testing"
)

func TestMissingPorts(t *testing.T) {
	out := "port 127.0.0.1:1080\nport 0.0.0.0:22\n"
	want := []string{"127.0.0.1:1080", "127.0.0.1:1081"}
	got := missingPorts(out, want)
	if len(got) != 1 || got[0] != "127.0.0.1:1081" {
		t.Fatalf("got %v, want only 127.0.0.1:1081 missing", got)
	}
	if got := missingPorts("port 127.0.0.1:9999\n", want); len(got) != 2 {
		t.Fatalf("unrelated port must not match, got %v", got)
	}
	if got := missingPorts(out, nil); len(got) != 0 {
		t.Fatalf("nothing wanted, nothing missing, got %v", got)
	}
}

func TestParseContainersAllHealthy(t *testing.T) {
	out := strings.Join([]string{
		"/app-app-1 running healthy",
		"/app-postgres-1 running healthy",
		"/app-support-1 running",
		"/app-egress-1 running",
		"port 127.0.0.1:1080",
	}, "\n")
	bad, seen := parseContainers(out)
	if len(bad) != 0 {
		t.Fatalf("want no bad containers, got %v", bad)
	}
	if seen != 4 {
		t.Fatalf("want seen=4, got %d", seen)
	}
}

func TestParseContainersDownAndUnhealthy(t *testing.T) {
	out := strings.Join([]string{
		"/app-app-1 exited",
		"/app-postgres-1 running unhealthy",
		"/app-support-1 running",
	}, "\n")
	bad, seen := parseContainers(out)
	if seen != 3 {
		t.Fatalf("want seen=3, got %d", seen)
	}
	want := []string{"app-app-1", "app-postgres-1"}
	if strings.Join(bad, ",") != strings.Join(want, ",") {
		t.Fatalf("got bad=%v, want %v", bad, want)
	}
}

func TestDecideNoProblemsNoPrev(t *testing.T) {
	msg, alert, st := decide(nil, false, "", 1000, 15, "Сервис")
	if msg != "" || alert || st != nil {
		t.Fatalf("expected silence, got msg=%q alert=%v st=%v", msg, alert, st)
	}
}

func TestDecideRecovery(t *testing.T) {
	prev := state{sum: "x", since: 1000, lastTime: 1000}
	msg, alert, st := decide(&prev, true, "", 1000+185, 15, "Сервис") // 3 min + a bit later
	if !alert || st != nil {
		t.Fatalf("recovery must alert and clear state, got alert=%v st=%v", alert, st)
	}
	if !strings.Contains(msg, "снова работает") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideNewOutage(t *testing.T) {
	msg, alert, st := decide(nil, false, "контейнер упал\n", 5000, 15, "Сервис")
	if !alert {
		t.Fatalf("a brand new outage must alert")
	}
	if st == nil || st.since != 5000 {
		t.Fatalf("since must be set to now for a new outage, got %+v", st)
	}
	if !strings.Contains(msg, "не работает") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideRepeatedWithinWindowIsSilent(t *testing.T) {
	problems := "контейнер упал\n"
	prev := state{sum: sumOf(problems), since: 1000, lastTime: 1000}
	msg, alert, st := decide(&prev, true, problems, 1000+5*60, 15, "Сервис") // 5 min later, window is 15
	if alert {
		t.Fatalf("same problem inside the repeat window must stay silent")
	}
	if st == nil || st.since != 1000 || st.lastTime != 1000 {
		t.Fatalf("state must be unchanged, got %+v", st)
	}
	if !strings.Contains(msg, "Уже сообщал") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideRepeatedAfterWindowAlertsAgain(t *testing.T) {
	problems := "контейнер упал\n"
	prev := state{sum: sumOf(problems), since: 1000, lastTime: 1000}
	now := int64(1000 + 20*60) // 20 min later, window is 15
	msg, alert, st := decide(&prev, true, problems, now, 15, "Сервис")
	if !alert {
		t.Fatalf("same problem after the repeat window must alert again")
	}
	if st == nil || st.since != 1000 || st.lastTime != now {
		t.Fatalf("since must be preserved, lastTime refreshed, got %+v", st)
	}
	if !strings.Contains(msg, "не работает") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestDecideChangedProblemAlertsImmediately(t *testing.T) {
	prev := state{sum: sumOf("A\n"), since: 1000, lastTime: 1000}
	msg, alert, st := decide(&prev, true, "B\n", 1000+60, 15, "Сервис") // 1 min later, well inside window
	if !alert {
		t.Fatalf("a changed problem signature must alert immediately")
	}
	if st == nil || st.since != 1000 {
		t.Fatalf("since (outage start) must be preserved across a changed problem, got %+v", st)
	}
	_ = msg
}

func TestStateRoundTrip(t *testing.T) {
	s := state{sum: "abc123", since: 111, lastTime: 222}
	parsed, ok := parseState(s.format())
	if !ok || parsed != s {
		t.Fatalf("round trip failed: got %+v ok=%v", parsed, ok)
	}
}

func TestParseStateMalformed(t *testing.T) {
	if _, ok := parseState("garbage"); ok {
		t.Fatalf("malformed state must not parse")
	}
}
