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

// flattenCSR: converts an adjacency‐list graph back into its CSR form
func FlattenCSR(G [][]int) (offs, edges []int) {
	n := len(G)
	offs = make([]int, n+1)
	for u := 0; u < n; u++ {
		offs[u+1] = offs[u] + len(G[u])
		edges = append(edges, G[u]...)
	}
	return
}
