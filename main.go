package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Read input test data and convert to adjacent list
func readAdjList(path string) [][]int {
	file, err := os.Open(path)
	if err != nil {
		panic(err) // Immediately abort the program and prints the error message
	}
	defer file.Close() // Schedules to run automatically when readAdjList returns, no matter it returns normally or panics

	var G [][]int                // Output graph (Adjacent list)
	sc := bufio.NewScanner(file) // Create a scanner to read the file line by line
	for sc.Scan() {              // Iterate over each non-empty line
		tok := strings.Fields(sc.Text()) // Split the line on spaces
		if len(tok) == 0 {               // Ignore empty lines in the text file
			continue
		}
		u, _ := strconv.Atoi(tok[0]) // First token is the source vertex u
		for len(G) <= u {            // Guarantee G is long enough so that index u exists
			G = append(G, nil)
		}
		for _, s := range tok[1:] {
			v, _ := strconv.Atoi(s)
			G[u] = append(G[u], v)
		}
	}
	// Final error check after scanning the file
	if err := sc.Err(); err != nil {
		panic(err)
	}
	return G
}

// Run the test
func main() {
	G := readAdjList("data/test.txt")
	// Create the transpose of G
	GT := make([][]int, len(G))
	for u, neighbors := range G {
		for _, v := range neighbors {
			GT[v] = append(GT[v], u)
		}
	}

	cbfs := &ClusterBFS{G: G, GT: GT, R: 2} // cbfs: a ClusterBFS object stored by pointer
	seeds := cbfs.Init([]int{0, 2})         // picks {0,2} as seeds
	cbfs.RunBFS(seeds)

	fmt.Println("Distances D:", cbfs.D)
	fmt.Println("Seed masks S:", cbfs.S)
	fmt.Println("G:", G)
	fmt.Println("GT:", GT)
}

/* Functions to be parallelized */
// AddVertices
// ToSeq
// CountTrue

// NewEdgeMap
// EdgeMapSparse, EdgeMapDense
// Run
