// server-check watches the server it runs on: disk, inodes, memory, swap, load,
// failed services, ssh/fail2ban, and the clock. Same anti-flutter behaviour as the old
// check.sh: the same set of problems is restated only every -repeat-hours, a new or changed
// one is reported at once.
//
// Exit 0 = nothing to tell the user. Exit 1 = print to stdout, the scheduler sends it.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envStr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func main() {
	dryRun := false
	for _, a := range os.Args[1:] {
		if a == "-dry-run" || a == "--dry-run" {
			dryRun = true
		}
	}

	th := defaultThresholds()
	th.diskMaxPercent = envInt("DISK_MAX_PERCENT", th.diskMaxPercent)
	th.inodesMaxPercent = envInt("INODES_MAX_PERCENT", th.inodesMaxPercent)
	th.memMinMB = envInt("MEM_MIN_MB", th.memMinMB)
	th.swapMaxPercent = envInt("SWAP_MAX_PERCENT", th.swapMaxPercent)
	th.swapLowMemMB = envInt("SWAP_LOW_MEM_MB", th.swapLowMemMB)
	th.swapinMaxPerSec = envInt("SWAPIN_MAX_PER_SEC", th.swapinMaxPerSec)
	th.loadMaxPerCPU = envInt("LOAD_MAX_PER_CPU", th.loadMaxPerCPU)
	th.repeatHours = envInt("REPEAT_HOURS", th.repeatHours)
	stateFile := envStr("STATE_FILE", "./data/last_alert")
	services := strings.Fields(envStr("SERVICES", "ssh fail2ban"))
	if _, err := exec.LookPath("caddy"); err == nil {
		services = append(services, "caddy")
	}

	var problems []string
	add := func(s string) { problems = append(problems, s) }

	if out, err := run("df", "-P", "/"); err == nil {
		if pct, err := parseDfPercent(out); err == nil && pct >= th.diskMaxPercent {
			add(fmt.Sprintf("Диск / заполнен на %d%% (порог %d%%)", pct, th.diskMaxPercent))
		}
	}

	if out, err := run("df", "-Pi", "/"); err == nil {
		if pct, err := parseDfPercent(out); err == nil && pct >= th.inodesMaxPercent {
			add(fmt.Sprintf("Кончаются места под файлы (inodes): %d%%", pct))
		}
	}

	meminfo, _ := os.ReadFile("/proc/meminfo")
	memMB, memErr := parseMemAvailableMB(string(meminfo))
	if memErr == nil && memMB < th.memMinMB {
		add(fmt.Sprintf("Доступно памяти %d МБ (порог %d МБ)", memMB, th.memMinMB))
	}

	if swapPct, err := swapPercent(string(meminfo)); err == nil && swapPct >= th.swapMaxPercent && memErr == nil && memMB < th.swapLowMemMB {
		add(fmt.Sprintf("Файл подкачки занят на %d%%, а свободной памяти %d МБ: памяти не хватает", swapPct, memMB))
	}

	vmstat1, _ := os.ReadFile("/proc/vmstat")
	in1, err1 := pswpin(string(vmstat1))
	time.Sleep(5 * time.Second)
	vmstat2, _ := os.ReadFile("/proc/vmstat")
	in2, err2 := pswpin(string(vmstat2))
	if err1 == nil && err2 == nil {
		swapin := (in2 - in1) / 5
		if swapin >= int64(th.swapinMaxPerSec) {
			add(fmt.Sprintf("Сервер постоянно достаёт данные из подкачки (%d стр./с): тормозит", swapin))
		}
	}

	cpus := runtime.NumCPU()
	loadavg, _ := os.ReadFile("/proc/loadavg")
	if over, load15, err := loadOverThreshold(string(loadavg), cpus, th.loadMaxPerCPU); err == nil && over {
		add(fmt.Sprintf("Нагрузка за 15 минут %.2f при %d ядрах", load15, cpus))
	}

	if out, err := run("systemctl", "--failed", "--no-legend", "--plain"); err == nil {
		var failed []string
		for _, line := range strings.Split(out, "\n") {
			if f := strings.Fields(line); len(f) > 0 {
				failed = append(failed, f[0])
			}
		}
		if len(failed) > 0 {
			add("Службы с ошибкой: " + strings.Join(failed, " "))
		}
	}

	for _, s := range services {
		if err := exec.Command("systemctl", "is-active", "--quiet", s).Run(); err != nil {
			add(fmt.Sprintf("Служба %s не работает", s))
		}
	}

	if out, err := run("timedatectl", "show", "-p", "NTPSynchronized", "--value"); err == nil {
		if strings.TrimSpace(out) == "no" {
			add("Часы сервера не сверяются с интернетом")
		}
	}

	problemText := strings.Join(problems, "\n")
	if problemText != "" {
		problemText += "\n"
	}

	now := time.Now().Unix()
	prevRaw, readErr := os.ReadFile(stateFile)
	prev, havePrev := state{}, false
	if readErr == nil {
		prev, havePrev = parseState(string(prevRaw))
	}

	message, alert, newState := decide(&prev, havePrev, problemText, now, th.repeatHours)

	if !dryRun {
		if newState == nil {
			os.Remove(stateFile)
		} else {
			os.MkdirAll("./data", 0o755)
			os.WriteFile(stateFile, []byte(newState.format()+"\n"), 0o644)
		}
	}

	if message != "" {
		fmt.Print(message)
		if !strings.HasSuffix(message, "\n") {
			fmt.Println()
		}
	}

	if dryRun {
		os.Exit(0)
	}
	if alert {
		os.Exit(1)
	}
	os.Exit(0)
}
