package parlay_go

import (
	"runtime"
	"sync"
)

// Helper function "parlay::pack_index" called by function "AddVertices" in ligra_light.go
func PackIndex(dense []bool) []int {
	n := len(dense)
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers

	locals := make([][]int, workers)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		lo := w * chunk
		hi := lo + chunk
		if hi > n {
			hi = n
		}
		if lo >= hi {
			workers = w
			break
		}
		wg.Add(1)
		// Launches a new goroutine
		go func(idx, lo, hi int) {
			defer wg.Done()
			var local []int
			// Each goroutine processes its chunk (find out all true values) and write the results to its corresponding "local"
			for i := lo; i < hi; i++ {
				if dense[i] {
					local = append(local, i)
				}
			}
			locals[idx] = local
		}(w, lo, hi)
	}
	wg.Wait()

	// Merge all locals
	total := 0
	for i := 0; i < workers; i++ {
		total += len(locals[i])
	}
	result := make([]int, 0, total)
	for i := 0; i < workers; i++ {
		result = append(result, locals[i]...)
	}
	return result
}
