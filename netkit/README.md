# netkit

CLI with two small tools: a system-info dump and a concurrent TCP port scanner.
Standard library only.

## Usage

### System info
```bash
go run . sysinfo
```
Prints OS, architecture, CPU count, hostname, local IPv4 addresses and the Go runtime version.

### Port scan
```bash
go run . scan -target 127.0.0.1 -start 1 -end 1024 [-workers 200] [-timeout 500]
```
- `-target` — host or IP to scan (required)
- `-start`, `-end` — port range (default 1-1024)
- `-workers` — concurrent workers (default 200)
- `-timeout` — per-port connect timeout in ms (default 500)

Uses a bounded worker pool over `net.DialTimeout`, so scan time is roughly
`(end-start)/workers * timeout` in the worst case (all ports closed/filtered).
