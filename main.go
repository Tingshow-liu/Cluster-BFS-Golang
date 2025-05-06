package main

import (
	"cluster_bfs_go/graphutils"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

func singleBatchTest(seeds [][]int, G, GT [][]int, t int, verify bool, R int, seq bool) { // par == True -> ClusterBFS; par == False -> Sequential BFS
	ns := len(seeds)
	k := len(seeds[0])
	// n := len(G)

	fmt.Printf("Radius: %d\n", R)
	fmt.Printf("Number of batches: %d, batch size k = %d\n", ns, k)

	// warm-up on first batch
	firstBatch := seeds[0]
	// Sequential BFS
	if seq {
		SequentialBFS(G, firstBatch)
	} else { // ClusterBFS
		cbfs := &ClusterBFS{G: G, GT: GT, R: R} // allocate ClusterBFS once
		goSeeds := cbfs.Init(firstBatch)
		cbfs.RunCBFS(goSeeds)
		if verify {
			if err := cbfs.VerifyCBFS(firstBatch); err != nil {
				fmt.Fprintf(os.Stderr, "verification failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("PASS correctness check!")
		}
	}

	// timed runs
	start := time.Now()
	if seq {
		for i := 0; i < t; i++ {
			for _, batch := range seeds {
				SequentialBFS(G, batch)
			}
			fmt.Printf("%d iteration done\n", i+1)
		}
	} else {
		cbfs := &ClusterBFS{G: G, GT: GT, R: R} // allocate ClusterBFS
		for i := 0; i < t; i++ {
			for _, batch := range seeds {
				goSeeds := cbfs.Init(batch)
				cbfs.RunCBFS(goSeeds)
			}
			fmt.Printf("%d iteration done\n", i+1)
		}
	}
	elapsed := time.Since(start)
	avg := elapsed / time.Duration(t)
	fmt.Printf("average cluster BFS time: %v\n", avg)
}

// Read the bin files and print part of the graph
func main() {
	// flags
	var (
		path   = flag.String("f", "", "path to graph.bin")
		t      = flag.Int("t", 3, "number of iterations")
		ns     = flag.Int("ns", 10, "number of seed batches")
		k      = flag.Int("k", 64, "seeds per batch")
		r      = flag.Int("r", 2, "BFS radius for verify")
		verify = flag.Bool("v", false, "verify with Ligra BFS")
		seq    = flag.Bool("seq", false, "if true, run ClusterBFS; if false, run Sequential BFS")
		c      = flag.Int("c", 20, "number of CPU cores to use (GOMAXPROCS)")
	)
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Usage: -f graph.bin [-t #] [-ns #] [-k #] [-r #] [-v] [-par=true|false]")
		os.Exit(1)
	}
	runtime.GOMAXPROCS(*c)

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
	singleBatchTest(seeds, G, GT, *t, *verify, *r, *seq)
}
