package main

/*
#cgo CXXFLAGS: -std=c++17
#cgo CXXFLAGS: -I${SRCDIR}/cwrapper
#cgo CXXFLAGS: -I${SRCDIR}/vendor/ligra
#cgo CXXFLAGS: -I${SRCDIR}/vendor/parlay
#cgo CXXFLAGS: -I${SRCDIR}/vendor/src
#cgo CXXFLAGS: -Wno-integer-overflow
#cgo CXXFLAGS: -Wno-shift-count-overflow
#cgo LDFLAGS: -lm

#cgo LDFLAGS: -lstdc++ cwrapper/wrapper.o

#include "cwrapper/wrapper.h"
*/
import "C" // To apply C++ Ligra code to Go for verification

import (
	"cluster_bfs_go/bitutils"
	"cluster_bfs_go/graphutils"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Member attributes of ClusterBFS
type ClusterBFS struct {
	G         [][]int // Input
	GT        [][]int // Input
	S0        []uint64
	S1        []uint64
	D         []uint64   // Output
	S         [][]uint64 // Output
	Distances []uint64
	R         int // Input
	INF       uint64
	round     uint64
}

// Initialize member attributes
func (cbfs *ClusterBFS) Init(vertices []int) []int {
	n := len(cbfs.G)      // Number of total vertices in graph G
	cbfs.INF = ^uint64(0) // Max uint64
	cbfs.S0 = make([]uint64, n)
	cbfs.S1 = make([]uint64, n)
	cbfs.D = make([]uint64, n) // Output of ClusterBFS
	cbfs.Distances = make([]uint64, n)
	cbfs.S = make([][]uint64, n)
	cbfs.round = 0

	// Zero initialize S, D, distances, S0, S1
	var wg sync.WaitGroup
	// Launch a goroutine for each i from 0 to n
	for i := 0; i < n; i++ {
		wg.Add(1) // Launch a new goroutine
		go func(i int) {
			defer wg.Done()
			cbfs.S0[i] = 0
			cbfs.S1[i] = 0
			cbfs.D[i] = cbfs.INF
			cbfs.Distances[i] = cbfs.INF
			cbfs.S[i] = make([]uint64, cbfs.R)
			for j := 0; j < cbfs.R; j++ {
				cbfs.S[i][j] = 0
			}
		}(i)
	}
	wg.Wait() // Wait for all goroutines to finish

	// Initialize the seed vertices (i.e., the starting points of BFS)
	seeds := []int{}
	for i, v := range vertices {
		if i != 0 && v == vertices[0] {
			break
		}
		cbfs.S1[v] = 1 << uint(i)
		seeds = append(seeds, v)
	}
	return seeds
}

// EdgeFunc: A bit-level parallel function, run by many threads (thread-level parallelism)
// EdgeFunc and CondFunc work together to let ligra know whether the current vertex can become one of the frontiers for the next level
func (cbfs *ClusterBFS) EdgeFunc(u, v int) bool {
	success := false
	// u tries to tell v what seeds visited u, so v can be reached by these seeds that visited u
	uVisited := atomic.LoadUint64(&cbfs.S0[u]) // seeds that reached u in earlier rounds
	vVisited := atomic.LoadUint64(&cbfs.S1[v]) // seeds already marked as reaching v in this round

	if (uVisited | vVisited) != vVisited {
		// some seeds that reached u haven't reached v yet
		bitutils.FetchOr(&cbfs.S1[v], uVisited)       // let v inherit those seed visits from u
		oldD := atomic.LoadUint64(&cbfs.Distances[v]) // read the value of distances[v] with an atomic operation (thread safe)
		// 1. if distances[v] == expected_val, atomically updates distances[v] to new_val
		// 2. Return true if the update happened; return false if any other thread changed it before this thread makes the change
		if oldD != cbfs.round {
			if atomic.CompareAndSwapUint64(&cbfs.Distances[v], oldD, cbfs.round) {
				success = true
			}
		}
	}
	return success
}

// FrontierFunc: runs after a vertex v that has been updated this round, and updates its records
func (cbfs *ClusterBFS) FrontierFunc(v int) {
	// S1[v] = all seeds that tried to reach v in this round
	// S0[v] = all seeds that had already reached v before this round
	// So difference = new seeds that just reached v this round
	difference := cbfs.S1[v] &^ cbfs.S0[v] // AND NOT (Guard)

	// If this is the first time v has been visited, set its BFS round (D[v])
	if cbfs.D[v] == cbfs.INF {
		cbfs.D[v] = cbfs.round
	}
	// S[v][r] stores which seeds reached v at relative round r;
	// -> round - D[v] (current round in BFS - the round when vertex v is first visited) gives that relative round number
	offset := cbfs.round - cbfs.D[v]
	cbfs.S[v][offset] = difference

	// Update S0[v] to include the new seeds — so in the next round, these won’t be counted again
	cbfs.S0[v] |= difference
}

// CondFunc: decides which vertices should be considered for updates in the current BFS round
// Returns true if:
// Vertices that haven’t been visited yet (D[v] == INF)
// Or were recently visited and cbfs.round-cbfs.D[v] is still within R from previous seeds
func (cbfs *ClusterBFS) CondFunc(v int, round uint64) bool {
	return cbfs.D[v] == cbfs.INF || (cbfs.round-cbfs.D[v]) < uint64(cbfs.R) // atomic version: dv := atomic.LoadUint64(&D[v])
}

// Test BFS within a single cluster
func (cbfs *ClusterBFS) RunCBFS(seeds []int) {
	// Initializes the initial frontiers of the BFS (cluster) from seeds
	frontier := NewEmptySparse()
	frontier.AddVertices(seeds)

	// getFunc: extract the destination vertex from an edge
	getFunc := func(e int) int {
		return e
	}

	// frontierMap: sets up the actual parallel BFS traversal
	frontierMap := NewEdgeMap(cbfs.G, cbfs.GT,
		func(u, v int, e int, backwards bool) bool {
			// just call your thread-safe edge logic (direction doesn't matter)
			return cbfs.EdgeFunc(u, v)
		},
		func(v int) bool {
			// ignore 'round' here because cbfs.round is global
			return cbfs.CondFunc(v, cbfs.round)
		},
		getFunc,
	)

	total := 0
	// Inner loop for BFS within the current frontiers
	for frontier.Size() > 0 {
		frontier.Apply(cbfs.FrontierFunc) // Update our output
		cbfs.round++
		m := frontier.Size()
		total += m
		frontier = frontierMap.Run(frontier, false) // Update the next level frontiers
	}
}

// VerifyCBFS: mimics the C++ verify_CBFS logic, using Ligra’s BFS via cgo
// seeds: the list of seed vertices (cbfs.Init returned these).
func (cbfs *ClusterBFS) VerifyCBFS(seeds []int) error {
	n := len(cbfs.G)
	if len(seeds) == 0 {
		return fmt.Errorf("no seeds provided")
	}

	// 1) Flatten G and GT into CSR form
	offsGo, edgesGo := graphutils.FlattenCSR(cbfs.G)
	offsGT, edgesGT := graphutils.FlattenCSR(cbfs.GT)

	// 2) allocate C-backed arrays
	offsC := make([]C.int, len(offsGo))
	edgesC := make([]C.int, len(edgesGo))
	for i, v := range offsGo {
		offsC[i] = C.int(v)
	}
	for i, v := range edgesGo {
		edgesC[i] = C.int(v)
	}

	// same for the transpose
	offsGTC := make([]C.int, len(offsGT))
	edgesGTC := make([]C.int, len(edgesGT))
	for i, v := range offsGT {
		offsGTC[i] = C.int(v)
	}
	for i, v := range edgesGT {
		edgesGTC[i] = C.int(v)
	}

	// 3) now call safely
	C.InitLigraGraph(
		(*C.int)(unsafe.Pointer(&offsC[0])), C.int(len(offsC)),
		(*C.int)(unsafe.Pointer(&edgesC[0])), C.int(len(edgesC)),
		(*C.int)(unsafe.Pointer(&offsGTC[0])), C.int(len(offsGTC)),
		(*C.int)(unsafe.Pointer(&edgesGTC[0])), C.int(len(edgesGTC)),
	)
	// Make sure to free memory at the end
	defer C.FreeLigraGraph()

	// Align Ligra's (C++) INF (2^31 - 1) with Go's INF (2^64 - 1)
	const ligraInf32 = (1 << 31) - 1
	R := cbfs.R

	// 3) For each seed, run Ligra BFS and compare
	for j, seed := range seeds {
		// stop if we cycle back to first seed
		if j != 0 && seed == seeds[0] {
			break
		}

		// prepare output buffer
		answer := make([]uint64, n)
		// call into C++ (no per-seed rebuild of G/GT)
		C.RunLigraBFS_CSR(
			C.int(seed),
			(*C.ulong)(unsafe.Pointer(&answer[0])),
		)

		// compare Ligra’s distances (answer) vs. your bit-parallel result
		for v := 0; v < n; v++ {
			dTrue := answer[v]
			dQuery := cbfs.D[v]
			if dTrue == ligraInf32 {
				// unreachable in true BFS, skip
				continue
			}
			// reconstruct the extra rounds from cbfs.S[v]
			var sum uint64
			changed := false
			for r := 0; r < R; r++ {
				sum |= cbfs.S[v][r]
				if sum&(1<<uint(j)) != 0 {
					dQuery += uint64(r)
					changed = true
					break
				}
			}
			// mismatch checks
			if changed {
				if dQuery != dTrue {
					return fmt.Errorf(
						"seed %d, vertex %d: true=%d, ours=%d",
						seed, v, dTrue, dQuery,
					)
				}
			} else {
				// allow up to ((R+1)/2)*2 slack
				if dTrue-dQuery > uint64((R+1)/2)*2 {
					return fmt.Errorf(
						"seed %d, vertex %d out of range: true=%d, ours=%d",
						seed, v, dTrue, dQuery,
					)
				}
			}
		}
	}
	return nil
}
