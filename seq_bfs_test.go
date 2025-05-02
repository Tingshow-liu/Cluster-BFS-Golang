package main

import (
	"cluster_bfs_go/graphutils"
	"flag"
	"fmt"
	"os"
	"testing"
)

// Input flags
var (
	path = flag.String("f", "", "path to graph.bin")
	k    = flag.Int("k", 8, "seeds per batch")
	ns   = flag.Int("ns", 1, "number of seed batches")
	r    = flag.Int("r", 4, "BFS radius for verify")
)

func TestMain(m *testing.M) {
	flag.Parse() // parse the flags *before* any TestXxx runs
	os.Exit(m.Run())
}

// *testing.T: the mechanism by which the test function communicates success or failure back to the Go test
func TestSequentialMatchesCluster(t *testing.T) {
	if *path == "" {
		fmt.Fprintln(os.Stderr, "Usage: -f graph.bin [-k #] [-ns #] [-r #]")
		os.Exit(1)
	}

	// Read the graph and select seeds
	offs64, edges32, err := graphutils.ReadGraphFromBin(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading graph: %v\n", err)
		os.Exit(1)
	}
	G := graphutils.BuildAdjFromCSR(offs64, edges32)
	GT := graphutils.TransposeAdj(G)

	// Select seeds (One batch is enough)
	seeds := make([][]int, *ns)
	for i := range seeds {
		seeds[i] = make([]int, *k)
	}
	graphutils.SelectSeeds1(G, seeds)
	firstBatch := seeds[0] // One batch is enough for testing!

	cbfs := &ClusterBFS{G: G, GT: GT, R: *r}
	goSeeds := cbfs.Init(firstBatch)
	cbfs.RunCBFS(goSeeds)

	Dseq, _ := SequentialBFS(G, firstBatch)
	for v := range G {
		if Dseq[v] != int(cbfs.D[v]) {
			t.Fatalf("v=%d: seq=%d vs cluster=%d", v, Dseq[v], cbfs.D[v])
		}
	}
}

/*
go test -v -run TestSequentialMatchesCluster -args -f data/graphs/Epinions1_sym.bin -k 3 -ns 1 -r 4
*/
