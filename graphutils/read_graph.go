package graphutils

import (
	"encoding/binary"
	"fmt"
	"os"
)

// ReadGraphFromBin read graph data from bin files "Sequentially" in the below format
/*
Data format:
n (uint64)
m (uint64)
sizes (uint64)
offsets[0…n] ( (n+1)×uint64 )
edgeIDs[0…m-1] ( m×uint32 )
*/
func ReadGraphFromBin(path string) (offsets []uint64, edges []uint32, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Read the three header words: n (number of vertices), m (number of edges), sizes
	var n, m, sizes uint64
	if err = binary.Read(f, binary.LittleEndian, &n); err != nil {
		return
	}
	if err = binary.Read(f, binary.LittleEndian, &m); err != nil {
		return
	}
	if err = binary.Read(f, binary.LittleEndian, &sizes); err != nil {
		return
	}
	// Print out the header values
	fmt.Printf("DEBUG: n=%d, m=%d, sizes=%d\n", n, m, sizes)
	// Sanity check: bytes for offsets + edges + header should match
	expected := (n+1)*8 + m*4 + 3*8
	if sizes != expected {
		return nil, nil, fmt.Errorf("size mismatch: got %d, expected %d", sizes, expected)
	}

	// Read n + 1 offsets (unit64 each)
	offsets = make([]uint64, n+1)
	if err = binary.Read(f, binary.LittleEndian, &offsets); err != nil {
		return
	}

	// Read m edge‑IDs (uint32 each)
	edges = make([]uint32, m)
	if err = binary.Read(f, binary.LittleEndian, &edges); err != nil {
		return
	}

	return offsets, edges, nil
}

// ReadBytePD loads the “bytepd” binary "Sequentially" in the below format (TBD)
/*
n (uint64)
m (uint64)
degree[0…n-1] (n×uint64): Number of neighbors of each vertex
edgeIDs[0…m-1] (m×uint64): Neighbor IDs of each vertex
*/
func ReadBytePD(path string) (offsets []uint64, edges []uint64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	// 1) Read header
	var n, m uint64
	if err = binary.Read(f, binary.LittleEndian, &n); err != nil {
		return
	}
	if err = binary.Read(f, binary.LittleEndian, &m); err != nil {
		return
	}
	fmt.Printf("DEBUG bytepd → n=%d, m=%d\n", n, m)

	// 2) Read per‑vertex byte counts
	degree := make([]uint64, n)
	if err = binary.Read(f, binary.LittleEndian, &degree); err != nil {
		return
	}

	// 3) Build CSR offsets by prefix‑sum
	offsets = make([]uint64, n+1)
	var sum uint64
	for i, d := range degree {
		offsets[i] = sum
		sum += d
	}
	offsets[n] = sum // should equal m

	// 4) Read neighbor IDs (as uint64)
	edges = make([]uint64, m)
	if err = binary.Read(f, binary.LittleEndian, &edges); err != nil {
		return
	}

	return offsets, edges, nil
}
