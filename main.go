package main

import (
	"cluster_bfs_go/graphutils"
	"flag"
	"fmt"
	"os"
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
		Dseq, _ := SequentialBFS(G, firstBatch)
		// DEBUG (TBD)
		reached := 0
		INF := 1_000_000_000
		for _, d := range Dseq {
			if d != INF { // since INF was set to R+1
				reached++
			}
		}
		fmt.Printf("Sequential BFS reached %d/%d vertices\n", reached, len(Dseq))
	} else { // ClusterBFS
		cbfs := &ClusterBFS{G: G, GT: GT, R: R} // allocate ClusterBFS once
		goSeeds := cbfs.Init(firstBatch)
		cbfs.RunCBFS(goSeeds)
		if verify {
			if err := cbfs.VerifyCBFS(firstBatch); err != nil {
				fmt.Fprintf(os.Stderr, "verification failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("PASS")
		}
		// DEBUG (TBD)
		reachedC := 0
		for _, d := range cbfs.D {
			if d != cbfs.INF {
				reachedC++
			}
		}
		fmt.Printf("Cluster BFS reached %d/%d vertices\n", reachedC, len(cbfs.D))
	}

	// timed runs
	start := time.Now()
	if seq {
		for i := 0; i < t; i++ {
			for _, batch := range seeds {
				SequentialBFS(G, batch)
			}
		}
	} else {
		cbfs := &ClusterBFS{G: G, GT: GT, R: R} // allocate ClusterBFS
		for i := 0; i < t; i++ {
			for _, batch := range seeds {
				goSeeds := cbfs.Init(batch)
				cbfs.RunCBFS(goSeeds)
				if verify {
					cbfs.VerifyCBFS(batch)
				}
			}
			fmt.Printf("%d iteration done\n", i)
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
	)
	flag.Parse()
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Usage: -f graph.bin [-t #] [-ns #] [-k #] [-r #] [-v] [-par=true|false]")
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
	singleBatchTest(seeds, G, GT, *t, *verify, *r, *seq)
}
