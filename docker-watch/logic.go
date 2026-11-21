package main

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

// missingPorts returns the wanted listen addresses (e.g. "127.0.0.1:1080") that do not show up
// in `ss -ltnH` output lines formatted as "port <addr>".
func missingPorts(out string, want []string) (missing []string) {
	have := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[0] == "port" {
			have[f[1]] = true
		}
	}
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	return
}

// parseContainers reads `docker inspect -f '{{.Name}} {{.State.Status}} {{.State.Health.Status}}'`
// output (one line per container, name prefixed with "/") and returns the names that are not
// running or are unhealthy, plus how many of the expected containers were seen at all.
func parseContainers(out string) (bad []string, seen int) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "/") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		seen++
		name := strings.TrimPrefix(fields[0], "/")
		status := ""
		health := ""
		if len(fields) > 1 {
			status = fields[1]
		}
		if len(fields) > 2 {
			health = fields[2]
		}
		if status != "running" || health == "unhealthy" {
			bad = append(bad, name)
		}
	}
	return
}

// state is the on-disk anti-flutter record: problem signature, when the outage started,
// and when it was last reported.
type state struct {
	sum      string
	since    int64
	lastTime int64
}

func parseState(raw string) (state, bool) {
	fields := strings.Fields(raw)
	if len(fields) != 3 {
		return state{}, false
	}
	since, err1 := strconv.ParseInt(fields[1], 10, 64)
	lastTime, err2 := strconv.ParseInt(fields[2], 10, 64)
	if err1 != nil || err2 != nil {
		return state{}, false
	}
	return state{sum: fields[0], since: since, lastTime: lastTime}, true
}

func (s state) format() string {
	return fmt.Sprintf("%s %d %d", s.sum, s.since, s.lastTime)
}

func sumOf(problems string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(problems)))[:32]
}

// decide is the anti-flutter logic: given the previous state (if any), the
// current problem text and "now", it returns what to print, whether that counts as an alert
// (exit 1 → the scheduler messages the user) and the new state to persist (nil = delete the file).
func decide(prev *state, havePrev bool, problems string, now int64, repeatMin int, name string) (message string, alert bool, newState *state) {
	if problems == "" {
		if havePrev {
			minutes := (now-prev.since)/60 + 1
			return fmt.Sprintf("✅ %s снова работает (лежал примерно %d мин)", name, minutes), true, nil
		}
		return "", false, nil
	}

	sum := sumOf(problems)
	since := now
	if havePrev {
		since = prev.since
		if prev.sum == sum && now-prev.lastTime < int64(repeatMin)*60 {
			return "Уже сообщал:\n" + problems, false, &state{sum: sum, since: since, lastTime: prev.lastTime}
		}
	}
	minutes := (now - since) / 60
	msg := fmt.Sprintf("🔴 %s не работает (%d мин):\n%s", name, minutes, problems)
	return msg, true, &state{sum: sum, since: since, lastTime: now}
}
