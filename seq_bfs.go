package main

// Sentry records “seed i reached vertex v at distance d”
type Sentry struct {
	Seed, Dist int
}

/* Without R!! */
// SequentialBFSWithS runs a plain multi‐source BFS from seeds that returns the same D and S as ClusterBFS
func SequentialBFS(G [][]int, seeds []int) (D []int, S [][]Sentry) {
	n := len(G)
	INF := 1_000_000_000

	// Initialize S and D
	D = make([]int, n)
	for i := range D {
		D[i] = INF
	}
	S = make([][]Sentry, n)

	// distBySeed[si][v] = best distance found so far from seed si to v
	distBySeed := make([][]int, len(seeds))
	for si := range seeds {
		distBySeed[si] = make([]int, n)
		for v := range distBySeed[si] {
			distBySeed[si][v] = INF
		}
	}

	// Queue items: (vertex v, seed index si, distance d)
	type Item struct{ v, si, d int }
	queue := make([]Item, 0, n*len(seeds))

	// Initialize the seeds
	for si, s := range seeds {
		if distBySeed[si][s] > 0 {
			distBySeed[si][s] = 0
			if D[s] > 0 {
				D[s] = 0
			}
			S[s] = append(S[s], Sentry{Seed: si, Dist: 0})
			queue = append(queue, Item{s, si, 0})
		}
	}

	// BFS over (v,si) pairs
	for head := 0; head < len(queue); head++ {
		curr := queue[head]
		u, si, d := curr.v, curr.si, curr.d
		nd := d + 1
		for _, v := range G[u] {
			// if this seed can reach v shorter than before (new info)
			if nd < distBySeed[si][v] {
				distBySeed[si][v] = nd
				// update the overall min-distance
				if nd < D[v] {
					D[v] = nd
				}
				// record this seed’s arrival
				S[v] = append(S[v], Sentry{Seed: si, Dist: nd})
				// enqueue for further propagation
				queue = append(queue, Item{v, si, nd})
			}
		}
	}

	return D, S
}

// /* Simple version (S[v] only contains a single entry point): TBD */
// func SequentialBFS_(G [][]int, seeds []int) (D []int, S [][]Sentry) {
// 	n := len(G)
// 	INF := 1_000_000_000

// 	// Initialize D and S
// 	D = make([]int, n)
// 	for i := range D {
// 		D[i] = INF
// 	}
// 	S = make([][]Sentry, n)

// 	// For each v, record which seed first reached it
// 	seedOf := make([]int, n)

// 	// Seed initialization
// 	queue := make([]int, 0, n)
// 	seen := make([]bool, n)
// 	for si, s := range seeds {
// 		if !seen[s] {
// 			seen[s] = true
// 			D[s] = 0
// 			seedOf[s] = si
// 			S[s] = append(S[s], Sentry{Seed: si, Dist: 0})
// 			queue = append(queue, s)
// 		}
// 	}

// 	// Standard FIFO BFS
// 	for head := 0; head < len(queue); head++ {
// 		u := queue[head]
// 		for _, v := range G[u] {
// 			if !seen[v] {
// 				seen[v] = true
// 				seedOf[v] = seedOf[u] // inherit the same seed
// 				D[v] = D[u] + 1
// 				S[v] = append(S[v], Sentry{Seed: seedOf[v], Dist: D[v]})
// 				queue = append(queue, v)
// 			}
// 		}
// 	}

// 	return D, S
// }
