package main

import (
	"cluster_bfs_go/bitutils" // go.mod is in "module cluster_bfs_go"
	"sync"
	"sync/atomic"
)

// Member attributes of ClusterBFS
type ClusterBFS struct {
	G         [][]int // The original graph
	GT        [][]int // The transpose of the graph
	S0        []uint64
	S1        []uint64
	D         []uint64
	S         [][]uint64
	Distances []uint64
	R         int // TBD
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
	difference := cbfs.S1[v] &^ cbfs.S0[v] // AND NOT

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
// Or were recently visited and cbfs.round-cbfs.D[v] is still within the radius R from previous seeds
func (cbfs *ClusterBFS) CondFunc(v int, round uint64) bool {
	return cbfs.D[v] == cbfs.INF || (cbfs.round-cbfs.D[v]) < uint64(cbfs.R) // atomic version: dv := atomic.LoadUint64(&D[v])
}

// Test BFS within a single cluster
func (cbfs *ClusterBFS) RunBFS(seeds []int) {
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
	// Inner loop for BFS within a single cluster.
	for frontier.Size() > 0 {
		frontier.Apply(cbfs.FrontierFunc)
		cbfs.round++
		m := frontier.Size()
		total += m
		frontier = frontierMap.Run(frontier, false)
	}
}
