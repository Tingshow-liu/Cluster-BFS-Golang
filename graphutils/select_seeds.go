package graphutils

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
)

// One-hop star
func SelectSeeds1(G [][]int, seeds [][]int) {
	n := len(G)
	setSize := len(seeds[0]) // Number of seeds in each batch
	// 1) make a random ordering of all vertices 0…n−1
	ord := rand.Perm(n)
	// 2) filter high-degree (vertices whose degree ≥ set_size)
	var verts []int
	for _, v := range ord {
		if len(G[v]) >= setSize {
			verts = append(verts, v)
		}
	}
	fmt.Printf("%d candidates\n", len(verts)) // Number of seed candidates
	// 3) Build the inverse map (getOrder) -> To make sure the fairness between all vertices to be selected as neighbors of the selected seeds
	getOrder := make([]int, n)
	for i, v := range ord {
		getOrder[v] = i
	}

	r := 0 // how many batches we’ve filled
	// 4–7) build each batch
	for _, v := range verts {
		seeds[r][0] = v
		ns := 1 // next free slot index
		// sort neighbors by getOrder[u] to impose the same randomness
		neigh := append([]int(nil), G[v]...)
		sort.Slice(neigh, func(i, j int) bool {
			// compare the two neighbor vertex IDs neigh[i] and neigh[j]
			// by looking up their positions in the global random order
			return getOrder[neigh[i]] < getOrder[neigh[j]]
		})
		for _, u := range neigh {
			// skip self-loops
			if u == v {
				continue
			}
			seeds[r][ns] = u
			ns++
			if ns == setSize { // If all slots are filled, stop
				break
			}
		}
		// pad if needed
		for ns < setSize {
			seeds[r][ns] = v
			ns++
		}
		// Move to the next seed-set row
		r++
		if r == len(seeds) {
			break
		}
	}
}

/* TBD */
// Two-hop star
func selectSeeds2(G [][]int, seeds [][]int) {
	n := len(G)
	setSize := len(seeds[0])
	// 1) random permutation of all vertices
	ord := rand.Perm(n)
	// 2) filter vertices with degree ≥ log(setSize)
	threshold := int(math.Log(float64(setSize)))
	var verts []int
	for _, v := range ord {
		if len(G[v]) >= threshold {
			verts = append(verts, v)
		}
	}
	// 3) build inverse map
	getOrder := make([]int, n)
	for i, v := range ord {
		getOrder[v] = i
	}
	// 4) for each center, collect up to setSize−1 from its 2-hop neighborhood
	r := 0
	for _, v := range verts {
		seeds[r][0] = v
		ns := 1
		// use a set to dedupe
		seen := make(map[int]struct{}, setSize)
		// 1-hop
		for _, u := range G[v] {
			if u != v {
				seen[u] = struct{}{}
			}
			// 2-hop
			for _, w := range G[u] {
				if w != v {
					seen[w] = struct{}{}
				}
			}
		}
		// turn set into slice
		neigh2 := make([]int, 0, len(seen))
		for u := range seen {
			neigh2 = append(neigh2, u)
		}
		// shuffle by same getOrder
		sort.Slice(neigh2, func(i, j int) bool {
			return getOrder[neigh2[i]] < getOrder[neigh2[j]]
		})
		// fill neighbors
		for _, u := range neigh2 {
			if ns == setSize {
				break
			}
			seeds[r][ns] = u
			ns++
		}
		// pad with center if needed
		for ns < setSize {
			seeds[r][ns] = v
			ns++
		}
		r++
		if r == len(seeds) {
			break
		}
	}
}

// Three-hop star
func selectSeeds3(G [][]int, seeds [][]int) {
	n := len(G)
	setSize := len(seeds[0])
	ord := rand.Perm(n)
	threshold := int(math.Log(float64(setSize)))
	var verts []int
	for _, v := range ord {
		if len(G[v]) >= threshold {
			verts = append(verts, v)
		}
	}
	getOrder := make([]int, n)
	for i, v := range ord {
		getOrder[v] = i
	}
	r := 0
	for _, src := range verts {
		seeds[r][0] = src
		ns := 1
		// visited array for 3-hop
		visited := make([]bool, n)
		visited[src] = true
		frontier := []int{src}
		// expand 3 times
		for hop := 0; hop < 3; hop++ {
			var next []int
			for _, u := range frontier {
				for _, w := range G[u] {
					if !visited[w] {
						visited[w] = true
						next = append(next, w)
					}
				}
			}
			frontier = next
		}
		// collect all visited ≠ src
		var neigh3 []int
		for u, ok := range visited {
			if ok && u != src {
				neigh3 = append(neigh3, u)
			}
		}
		// shuffle by getOrder
		sort.Slice(neigh3, func(i, j int) bool {
			return getOrder[neigh3[i]] < getOrder[neigh3[j]]
		})
		// fill
		for _, u := range neigh3 {
			if ns == setSize {
				break
			}
			seeds[r][ns] = u
			ns++
		}
		for ns < setSize {
			seeds[r][ns] = src
			ns++
		}
		r++
		if r == len(seeds) {
			break
		}
	}
}
