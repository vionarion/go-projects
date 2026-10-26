// netkit is a small CLI with two tools a DevOps engineer reaches for often:
// a system-info dump and a concurrent TCP port scanner.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "sysinfo":
		if err := runSysinfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "scan":
		scanCmd := flag.NewFlagSet("scan", flag.ExitOnError)
		target := scanCmd.String("target", "", "Host or IP to scan (required)")
		start := scanCmd.Int("start", 1, "First port in range")
		end := scanCmd.Int("end", 1024, "Last port in range")
		workers := scanCmd.Int("workers", 200, "Number of concurrent workers")
		timeoutMs := scanCmd.Int("timeout", 500, "Per-port connect timeout in ms")
		scanCmd.Parse(os.Args[2:])

		opts := scanOptions{
			target:  *target,
			start:   *start,
			end:     *end,
			workers: *workers,
			timeout: *timeoutMs,
		}
		open, err := runScan(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(open) == 0 {
			fmt.Println("No open ports found")
			return
		}
		for _, p := range open {
			fmt.Printf("%d/tcp open\n", p)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: netkit <sysinfo|scan> [options]")
	fmt.Println("\nSystem info:")
	fmt.Println("  netkit sysinfo")
	fmt.Println("\nPort scan:")
	fmt.Println("  netkit scan -target 127.0.0.1 -start 1 -end 1024 [-workers 200] [-timeout 500]")
}
