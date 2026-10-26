# netkit

CLI with small tools for day-to-day server work: a system-info dump, a
concurrent TCP port scanner, and a disk-hog finder. Standard library only,
cross-platform.

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

### Largest files
```bash
go run . bigfiles -root /var -top 10
```
- `-root` — directory to scan (default `.`)
- `-top` — number of largest files to show (default 10)

Walks the tree once, keeping the N largest files in a min-heap — no need to
sort or hold every file in memory. Unreadable paths are skipped.
