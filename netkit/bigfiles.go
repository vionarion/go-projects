package main

import (
	"container/heap"
	"fmt"
	"io/fs"
	"path/filepath"
)

type fileEntry struct {
	path string
	size int64
}

// minHeap keeps the N largest files seen so far by evicting the smallest.
type minHeap []fileEntry

func (h minHeap) Len() int            { return len(h) }
func (h minHeap) Less(i, j int) bool  { return h[i].size < h[j].size }
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(fileEntry)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	last := old[n-1]
	*h = old[:n-1]
	return last
}

// topNLargestFiles walks root and returns the n largest regular files,
// largest first. Unreadable paths are skipped rather than aborting the walk.
func topNLargestFiles(root string, n int) ([]fileEntry, error) {
	h := &minHeap{}
	heap.Init(h)

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		entry := fileEntry{path: path, size: info.Size()}
		if h.Len() < n {
			heap.Push(h, entry)
		} else if h.Len() > 0 && entry.size > (*h)[0].size {
			heap.Pop(h)
			heap.Push(h, entry)
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	result := make([]fileEntry, h.Len())
	for i := len(result) - 1; i >= 0; i-- {
		result[i] = heap.Pop(h).(fileEntry)
	}
	return result, nil
}

func runBigfiles(root string, n int) error {
	files, err := topNLargestFiles(root, n)
	if err != nil {
		return err
	}
	for _, f := range files {
		fmt.Printf("%8.2f MB  %s\n", float64(f.size)/(1<<20), f.path)
	}
	return nil
}
