package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

// SystemInfo holds the facts sysinfo collects. It is kept small and
// cross-platform: anything that needs OS-specific syscalls (e.g. total RAM)
// is deliberately left out rather than faked on platforms that lack it.
type SystemInfo struct {
	OS        string
	Arch      string
	CPUs      int
	Hostname  string
	LocalIPs  []string
	GoVersion string
}

func collectSystemInfo() (SystemInfo, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("hostname: %w", err)
	}

	ips, err := localIPs()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("local IPs: %w", err)
	}

	return SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		Hostname:  hostname,
		LocalIPs:  ips,
		GoVersion: runtime.Version(),
	}, nil
}

// localIPs returns non-loopback IPv4 addresses of the host's network interfaces.
func localIPs() ([]string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			ips = append(ips, ip4.String())
		}
	}
	return ips, nil
}

func runSysinfo() error {
	info, err := collectSystemInfo()
	if err != nil {
		return err
	}

	fmt.Printf("OS:         %s\n", info.OS)
	fmt.Printf("Arch:       %s\n", info.Arch)
	fmt.Printf("CPUs:       %d\n", info.CPUs)
	fmt.Printf("Hostname:   %s\n", info.Hostname)
	fmt.Printf("Local IPs:  %v\n", info.LocalIPs)
	fmt.Printf("Go runtime: %s\n", info.GoVersion)
	return nil
}
