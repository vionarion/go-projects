// docker-watch checks over ssh that the given Docker containers on a remote host are running
// and healthy, that the given local ports on it are listening, and (optionally) touches a
// heartbeat file on a second host so that a backup watchdog there knows this one is alive.
// A lasting problem is repeated only every REPEAT_MIN minutes; a changed or recovered state is
// reported at once.
//
// Exit 0 = nothing to tell the user. Exit 1 = print to stdout, the scheduler sends it.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func env(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func envInt(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func runSSH(timeout time.Duration, host string, remoteCmd string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", host, remoteCmd)
	out, err := cmd.Output()
	return string(out), err
}

func probe(host string, containers []string) string {
	remote := fmt.Sprintf(
		"docker inspect -f '{{.Name}} {{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' %s; ss -ltnH | awk '{print \"port\", $4}'",
		strings.Join(containers, " "))
	out, err := runSSH(25*time.Second, host, remote)
	if err != nil {
		return ""
	}
	return out
}

func main() {
	dryRun := false
	for _, a := range os.Args[1:] {
		if a == "-dry-run" || a == "--dry-run" {
			dryRun = true
		}
	}

	host := os.Getenv("WATCH_HOST") // user@host, reachable by ssh key
	containers := strings.Fields(os.Getenv("WATCH_CONTAINERS"))
	if host == "" || len(containers) == 0 {
		fmt.Fprintln(os.Stderr, "set WATCH_HOST and WATCH_CONTAINERS (see README)")
		os.Exit(2)
	}
	name := env("WATCH_NAME", "Сервис")
	ports := strings.Fields(os.Getenv("WATCH_PORTS"))
	heartbeatHost := os.Getenv("HEARTBEAT_HOST")
	heartbeatCmd := env("HEARTBEAT_CMD", "touch ~/watch-heartbeat")
	repeatMin := envInt("REPEAT_MIN", 15)
	stateFile := env("STATE_FILE", "./data/docker_watch")

	var problems []string
	add := func(s string) { problems = append(problems, s) }

	if heartbeatHost != "" {
		if _, err := runSSH(20*time.Second, heartbeatHost, heartbeatCmd); err != nil {
			add("Сервер для сигнала жизни (" + heartbeatHost + ") не отвечает")
		}
	}

	out := probe(host, containers)
	if out == "" {
		time.Sleep(10 * time.Second) // one network blip is not an outage
		out = probe(host, containers)
	}

	if out == "" {
		add("Сервер " + host + " не отвечает (или Docker на нём не работает)")
	} else {
		if missing := missingPorts(out, ports); len(missing) > 0 {
			add("Не слушают порты: " + strings.Join(missing, " "))
		}
		bad, seen := parseContainers(out)
		if len(bad) > 0 {
			add("Не работают контейнеры: " + strings.Join(bad, " "))
		}
		if seen < len(containers) {
			add("Пропали контейнеры: найдено " + strconv.Itoa(seen) + " из " + strconv.Itoa(len(containers)))
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

	message, alert, newState := decide(&prev, havePrev, problemText, now, repeatMin, name)

	if !dryRun {
		if newState == nil {
			os.Remove(stateFile)
		} else {
			os.MkdirAll(filepath.Dir(stateFile), 0o755)
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
