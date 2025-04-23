package main

import (
	"cluster_bfs_go/graphutils"
	"fmt"
	"os"
)

// Read the bin files and print part of the graph
func main() {
	// Read input data and construct the graph
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s graph.bin\n", os.Args[0])
		os.Exit(1)
	}
	path := os.Args[1]

	// start := time.Now()
	offsets, edges, err := graphutils.ReadGraphFromBin(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading graph: %v\n", err)
		os.Exit(1)
	}
	// fmt.Printf("Load time: %v\n\n", time.Since(start))

	// Print first 5 vertices (or fewer, if n<5)
	n := len(offsets) - 1
	limit := 5
	if n < limit {
		limit = n
	}
	for i := 0; i < limit; i++ {
		startIdx := offsets[i]
		endIdx := offsets[i+1]
		neighbors := edges[startIdx:endIdx]
		fmt.Printf("Vertex %3d has %2d neighbors: %v\n",
			i, len(neighbors), neighbors)
	}

	// Seeds selection
}

// // Run the test for our own tiny graph (data/test.txt) (TBD)
// func main() {
// 	G := utils.ReadAdjList("data/test.txt")
// 	// Create the transpose of G
// 	GT := make([][]int, len(G))
// 	for u, neighbors := range G {
// 		for _, v := range neighbors {
// 			GT[v] = append(GT[v], u)
// 		}
// 	}

// 	cbfs := &ClusterBFS{G: G, GT: GT, R: 2} // cbfs: a ClusterBFS object stored by pointer
// 	seeds := cbfs.Init([]int{0, 2})         // picks {0,2} as seeds
// 	cbfs.RunBFS(seeds)

// 	fmt.Println("Distances D:", cbfs.D)
// 	fmt.Println("Seed masks S:", cbfs.S)
// 	fmt.Println("G:", G)
// 	fmt.Println("GT:", GT)
// }
