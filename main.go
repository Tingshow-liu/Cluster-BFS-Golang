package main

import (
	"cluster_bfs_go/graphutils"
	"flag"
	"fmt"
	"os"
)

func singleBatchTest(seeds [][]int, G, GT [][]int, t int, verify bool, R int) {
	ns := len(seeds)
	k := len(seeds[0])
	// n := len(G)

	fmt.Printf("Radius: %d\n", R)
	fmt.Printf("Number of batches: %d, batch size k = %d\n", ns, k)

	// allocate ClusterBFS once
	cbfs := &ClusterBFS{G: G, GT: GT, R: R}

	// warm-up on first batch
	firstBatch := seeds[0]
	goSeeds := cbfs.Init(firstBatch)
	cbfs.RunCBFS(goSeeds)
	if verify {
		cbfs.VerifyCBFS(firstBatch)
	}

	// // timed runs
	// start := time.Now()
	// for i := 0; i < t; i++ {
	// 	for _, batch := range seeds {
	// 		goSeeds = cbfs.Init(batch)
	// 		cbfs.RunCBFS(goSeeds)
	// 		if verify {
	// 			cbfs.VerifyCBFS(first)
	// 		}
	// 	}
	// }
	// elapsed := time.Since(start)
	// avg := elapsed / time.Duration(t)
	// fmt.Printf("average cluster BFS time: %v\n", avg)

}

// Read the bin files and print part of the graph
func main() {
	// flags
	var (
		path   = flag.String("f", "", "path to graph.bin")
		t      = flag.Int("t", 3, "number of iterations")
		ns     = flag.Int("ns", 10, "number of seed batches")
		k      = flag.Int("k", 8, "seeds per batch")
		r      = flag.Int("r", 4, "BFS radius for verify")
		verify = flag.Bool("v", false, "verify with Ligra BFS")
	)
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Usage: -f graph.bin [-t #] [-ns #] [-k #] [-v]")
		os.Exit(1)
	}

	// Read CSR and construct the graph
	offs64, edges32, err := graphutils.ReadGraphFromBin(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading graph: %v\n", err)
		os.Exit(1)
	}

	// Build Go adjacent lists
	G := graphutils.BuildAdjFromCSR(offs64, edges32)
	GT := graphutils.TransposeAdj(G)

	// Select seeds
	seeds := make([][]int, *ns)
	for i := range seeds {
		seeds[i] = make([]int, *k)
	}
	graphutils.SelectSeeds1(G, seeds)

	// run single‐batch test
	singleBatchTest(seeds, G, GT, *t, *verify, *r)
}
