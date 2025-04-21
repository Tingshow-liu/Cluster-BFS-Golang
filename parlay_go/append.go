package parlay_go

import (
	"runtime"
	"sync"
)

// Helper function "parlay::append" called by function "AddVertices" in ligra_light.go
func Append(src []int, dst []int) {
	n := len(src)
	if n == 0 { // To avoid "integer divide by zero" when calculating chunk later
		return
	}
	// Decide number of workes, which is the number of logical CPUs
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	// Compute chunk size (Number of elements processed by each goroutine)
	chunk := (n + workers - 1) / workers

	// Let all goroutines run in parallel
	var wg sync.WaitGroup
	// Create worker goroutines to do the copy task (copy(d, s)) in parallel (Creation of goroutines is sequential, but multiple goroutines run in parallel to do the copy jobs)
	for i := 0; i < workers; i++ {
		start := i * chunk
		end := start + chunk
		if end > n {
			end = n
		}
		if start >= end {
			break
		}
		wg.Add(1)
		// Launches a new goroutine
		go func(s, d []int) {
			defer wg.Done() // Decrements the WaitGroup counter—signaling “this piece of work is finished.
			copy(d, s)      // Copies all elements from slice s into slice d
		}(src[start:end], dst[start:end])
	}
	// Wait for all copies to finish
	wg.Wait()
}
