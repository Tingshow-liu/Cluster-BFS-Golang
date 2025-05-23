package graphutils

// BuildAdjFromCSR: turns CSR into an adjacency‐list [][]int
func BuildAdjFromCSR(offsets []uint64, edges []uint32) [][]int {
	n := len(offsets) - 1
	G := make([][]int, n)
	for u := 0; u < n; u++ {
		for idx := offsets[u]; idx < offsets[u+1]; idx++ {
			G[u] = append(G[u], int(edges[idx]))
		}
	}
	return G
}

// TransposeAdj: transposes G (adjacent list) to GT
func TransposeAdj(G [][]int) [][]int {
	n := len(G)
	GT := make([][]int, n)
	for u, nbrs := range G {
		for _, v := range nbrs {
			GT[v] = append(GT[v], u)
		}
	}
	return GT
}

// FlattenCSR takes an adjacency list and produces
// the CSR offsets+edges arrays.
func FlattenCSR(G [][]int) ([]int, []int) {
	n := len(G)
	// make the offsets array one longer than the number of vertices
	offs := make([]int, n+1)

	// (optional) pre-allocate edges so we don’t keep growing the slice
	total := 0
	for _, nbrs := range G {
		total += len(nbrs)
	}
	edges := make([]int, 0, total)

	// build prefix sums and edge list
	for u, nbrs := range G {
		offs[u+1] = offs[u] + len(nbrs)
		edges = append(edges, nbrs...)
	}

	// now *actually* return the two slices
	return offs, edges
}
