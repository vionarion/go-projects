package main

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

type thresholds struct {
	diskMaxPercent   int
	inodesMaxPercent int
	memMinMB         int
	swapMaxPercent   int
	swapLowMemMB     int
	swapinMaxPerSec  int
	loadMaxPerCPU    int
	repeatHours      int
}

func defaultThresholds() thresholds {
	return thresholds{
		diskMaxPercent:   85,
		inodesMaxPercent: 85,
		memMinMB:         200,
		swapMaxPercent:   80,
		swapLowMemMB:     500,
		swapinMaxPerSec:  200,
		loadMaxPerCPU:    2,
		repeatHours:      6,
	}
}

// parseDfPercent reads the "Use%"/"IUse%" column of `df -P` / `df -Pi` output for the second line.
func parseDfPercent(out string) (int, error) {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return 0, fmt.Errorf("unexpected df fields: %q", lines[1])
	}
	pct := strings.TrimSuffix(fields[4], "%")
	return strconv.Atoi(pct)
}

// parseMemAvailableMB reads MemAvailable from /proc/meminfo content, in MB.
func parseMemAvailableMB(meminfo string) (int, error) {
	for _, line := range strings.Split(meminfo, "\n") {
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				break
			}
			kb, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}
			return kb / 1024, nil
		}
	}
	return 0, fmt.Errorf("MemAvailable not found")
}

// swapPercent returns how full swap is, 0 if there is no swap at all.
func swapPercent(meminfo string) (int, error) {
	var total, free int64
	haveTotal, haveFree := false, false
	for _, line := range strings.Split(meminfo, "\n") {
		if strings.HasPrefix(line, "SwapTotal:") {
			if f := strings.Fields(line); len(f) >= 2 {
				total, _ = strconv.ParseInt(f[1], 10, 64)
				haveTotal = true
			}
		} else if strings.HasPrefix(line, "SwapFree:") {
			if f := strings.Fields(line); len(f) >= 2 {
				free, _ = strconv.ParseInt(f[1], 10, 64)
				haveFree = true
			}
		}
	}
	if !haveTotal || !haveFree {
		return 0, fmt.Errorf("swap fields not found")
	}
	if total == 0 {
		return 0, nil
	}
	return int((total - free) * 100 / total), nil
}

func pswpin(vmstat string) (int64, error) {
	for _, line := range strings.Split(vmstat, "\n") {
		if strings.HasPrefix(line, "pswpin ") {
			f := strings.Fields(line)
			if len(f) < 2 {
				break
			}
			return strconv.ParseInt(f[1], 10, 64)
		}
	}
	return 0, fmt.Errorf("pswpin not found")
}

func loadOverThreshold(loadavg string, cpus, maxPerCPU int) (bool, float64, error) {
	fields := strings.Fields(loadavg)
	if len(fields) < 2 {
		return false, 0, fmt.Errorf("unexpected /proc/loadavg: %q", loadavg)
	}
	load15, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return false, 0, err
	}
	return load15 > float64(cpus*maxPerCPU), load15, nil
}

type state struct {
	sum  string
	time int64
}

func parseState(raw string) (state, bool) {
	fields := strings.Fields(raw)
	if len(fields) != 2 {
		return state{}, false
	}
	t, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return state{}, false
	}
	return state{sum: fields[0], time: t}, true
}

func (s state) format() string {
	return fmt.Sprintf("%s %d", s.sum, s.time)
}

func sumOf(problems string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(problems)))[:32]
}

// decide mirrors check.sh: a new or changed set of problems is reported (alert=true) and
// remembered; the same set within repeatHours is restated but silent; no problems clears state.
func decide(prev *state, havePrev bool, problems string, now int64, repeatHours int) (message string, alert bool, newState *state) {
	if problems == "" {
		return "", false, nil
	}
	sum := sumOf(problems)
	if havePrev && prev.sum == sum && now-prev.time < int64(repeatHours)*3600 {
		msg := fmt.Sprintf("Те же проблемы, что и раньше; уже сообщал, напомню через %d ч:\n%s", repeatHours, problems)
		return msg, false, prev
	}
	return problems, true, &state{sum: sum, time: now}
}
