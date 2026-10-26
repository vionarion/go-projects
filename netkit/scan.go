package main

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"time"
)

type scanOptions struct {
	target  string
	start   int
	end     int
	workers int
	timeout int // milliseconds
}

func (o scanOptions) validate() error {
	if o.target == "" {
		return fmt.Errorf("target is required")
	}
	if o.start < 1 || o.end > 65535 || o.start > o.end {
		return fmt.Errorf("invalid port range %d-%d", o.start, o.end)
	}
	if o.workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}
	return nil
}

// runScan scans o.target over the given port range with a bounded pool of
// workers and returns the open ports in ascending order.
func runScan(o scanOptions) ([]int, error) {
	if err := o.validate(); err != nil {
		return nil, err
	}

	timeout := time.Duration(o.timeout) * time.Millisecond
	ports := make(chan int)
	var open []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < o.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range ports {
				addr := net.JoinHostPort(o.target, fmt.Sprintf("%d", port))
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err != nil {
					continue
				}
				conn.Close()
				mu.Lock()
				open = append(open, port)
				mu.Unlock()
			}
		}()
	}

	for p := o.start; p <= o.end; p++ {
		ports <- p
	}
	close(ports)
	wg.Wait()

	sort.Ints(open)
	return open, nil
}
