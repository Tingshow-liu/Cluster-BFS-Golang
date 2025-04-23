package graphutils

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// Read input test data and convert to adjacent list (TBD)
func ReadAdjList(path string) [][]int {
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
