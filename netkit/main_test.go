package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestScanOptionsValidate(t *testing.T) {
	tests := []struct {
		name      string
		opts      scanOptions
		expectErr bool
	}{
		{"valid", scanOptions{target: "127.0.0.1", start: 1, end: 1024, workers: 10, timeout: 100}, false},
		{"missing target", scanOptions{target: "", start: 1, end: 1024, workers: 10, timeout: 100}, true},
		{"start after end", scanOptions{target: "127.0.0.1", start: 100, end: 1, workers: 10, timeout: 100}, true},
		{"port out of range", scanOptions{target: "127.0.0.1", start: 1, end: 70000, workers: 10, timeout: 100}, true},
		{"no workers", scanOptions{target: "127.0.0.1", start: 1, end: 1024, workers: 0, timeout: 100}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("validate() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestRunScanFindsOpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	open, err := runScan(scanOptions{
		target:  "127.0.0.1",
		start:   port,
		end:     port,
		workers: 5,
		timeout: 500,
	})
	if err != nil {
		t.Fatalf("runScan: %v", err)
	}
	if len(open) != 1 || open[0] != port {
		t.Errorf("expected [%d], got %v", port, open)
	}
}

func TestTopNLargestFiles(t *testing.T) {
	dir := t.TempDir()
	sizes := map[string]int{"small.txt": 10, "medium.txt": 100, "big.txt": 1000}
	for name, size := range sizes {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	files, err := topNLargestFiles(dir, 2)
	if err != nil {
		t.Fatalf("topNLargestFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if filepath.Base(files[0].path) != "big.txt" || filepath.Base(files[1].path) != "medium.txt" {
		t.Errorf("expected [big.txt medium.txt] in order, got %v", files)
	}
}

func TestCollectSystemInfo(t *testing.T) {
	info, err := collectSystemInfo()
	if err != nil {
		t.Fatalf("collectSystemInfo: %v", err)
	}
	if info.OS == "" || info.Hostname == "" || info.CPUs < 1 {
		t.Errorf("unexpected empty/zero fields: %+v", info)
	}
}
