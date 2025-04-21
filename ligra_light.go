package main

// analogue to Parlay's parallel loops
import (
	"sync"
	"sync/atomic"
	"runtime"
)

// ----------------------------------------------------
// VertexSubset: a lightweight representation of a set
// of vertices in either "sparse" or "dense" format.
//
// We assume vertices are of type "int" so that dense
// representation (a slice of bool) uses the vertex as an index.
// ----------------------------------------------------

// VertexSubset represents a subset of vertices.
type VertexSubset struct {
	// isSparse indicates whether the subset is stored in a sparse form.
	isSparse bool
	// n is the number of vertices in the subset.
	n int
	// sparse holds the list of vertices when stored sparsely.
	sparse []int
	// dense holds a boolean slice where each index indicates membership.
	dense []bool
}

// Size returns the number of vertices in the subset.
func (vs *VertexSubset) Size() int {
	return vs.n
}

// NewEmptySparse creates a new, empty sparse vertex subset.
func NewEmptySparse() VertexSubset {
	return VertexSubset{isSparse: true, n: 0, sparse: []int{}}
}

// NewSparse creates a vertex subset from an existing slice of vertices.
func NewSparse(vertices []int) VertexSubset {
	return VertexSubset{isSparse: true, n: len(vertices), sparse: vertices}
}

// NewSingle creates a vertex subset containing a single vertex.
func NewSingle(v int) VertexSubset {
	return VertexSubset{isSparse: true, n: 1, sparse: []int{v}}
}

// NewDense creates a vertex subset from a provided dense boolean slice.
func NewDense(dense []bool) VertexSubset {
	// Count the number of true values.
	// count := 0
	// for _, b := range dense {
	// 	if b {
	// 		count++
	// 	}
	// }
	count := countTrue(dense)
	return VertexSubset{isSparse: false, n: count, dense: dense}
}

// AddVertices adds a list of vertices to the subset.
// If the representation is sparse, it appends the new vertices;
// if dense, it sets the corresponding indices to true.
func (vs *VertexSubset) AddVertices(V []int) {
	if vs.isSparse {
		vs.sparse = append(vs.sparse, V...)
		vs.n += len(V)
	} else {
		for _, v := range V {
			vs.dense[v] = true
		}
	}
	vs.n = countTrue(vs.dense)
}

// ToSeq returns a unified slice of vertices regardless
// of the current representation.
func (vs *VertexSubset) ToSeq() []int {
	if vs.isSparse {
		return vs.sparse
	}
	var seq []int
	for i, active := range vs.dense {
		if active {
			seq = append(seq, i)
		}
	}
	return seq
}

// Apply applies a function f to every vertex in the subset.
// For the sparse representation, it spawns a goroutine per vertex.
// For the dense representation, it iterates over the boolean slice.
func (vs *VertexSubset) Apply(f func(int)) {
	var wg sync.WaitGroup
	if vs.isSparse {
		for _, v := range vs.sparse {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				f(val)
			}(v)
		}
	} else {
		for i, active := range vs.dense {
			if active {
				wg.Add(1)
				go func(val int) {
					defer wg.Done()
					f(val)
				}(i)
			}
		}
	}
	wg.Wait()
}

// ----------------------------------------------------
// identity: an identity function implementation.
// In Go, this can simply be a generic function.
// ----------------------------------------------------

// Identity returns its input unchanged.
func Identity[T any](x T) T {
	return x
}
// func IdentityGet[E any](e E) int { return e.(int) }

// ----------------------------------------------------
// EdgeMap: implements an edge mapping similar to the Ligra
// interface. The EdgeMap operates on a graph represented
// as slices of slices of edges. Each vertex in the graph is
// an index (int) and each edge can be any type.
// The fields:
//    - n: number of vertices,
//    - m: total number of edges (computed during construction),
//    - fa: the function applied to live edges,
//    - get: extracts a vertex from an edge (defaults to identity),
//    - cond: a condition that tests if a vertex meets a criterion,
//    - G: the forward graph (adjacency lists),
//    - GT: the transposed graph for backward traversals.
// ----------------------------------------------------

// EdgeMap represents the edge mapping structure.
// Here E is the edge type, and we assume vertices are int.
type EdgeMap[E any] struct {
	n    int                                      // number of vertices
	m    int64                                    // total edges (64-bit for atomic ops)
	fa   func(u, v int, e E, backwards bool) bool // processing function for each edge
	get  func(e E) int                            // extracts a vertex from an edge; identity by default
	cond func(v int) bool                         // condition to test if vertex v qualifies
	G    [][]E                                    // forward graph (adjacency list for each vertex)
	GT   [][]E                                    // transposed graph for backward traversal
}

// NewEdgeMap constructs a new EdgeMap. It computes n (number of vertices)
// and m (total number of edges) from the input graph G.
func NewEdgeMap[E any](G, GT [][]E,
    fa func(u, v int, e E, backwards bool) bool,
    cond func(v int) bool,
    get func(e E) int) *EdgeMap[E] {
    n := len(G)
    // Initialize m as atomic int64
    var m int64
    var wg sync.WaitGroup
    // Parallel accumulate edge counts
    for _, edges := range G {
        wg.Add(1)
        go func(es []E) {
            defer wg.Done()
            // atomically add the length of this slice to m
            atomic.AddInt64(&m, int64(len(es)))
        }(edges)
    }
    wg.Wait()

    return &EdgeMap[E]{
        n:    n,
        m:    m,
        fa:   fa,
        get:  get,
        cond: cond,
        G:    G,
        GT:   GT,
    }
}


// f is a wrapper that calls the user-provided function fa with four arguments.
// In the original C++ code, a compile-time conditional was used to adjust parameters,
// but in Go we expect fa always to have the signature with all parameters.
func (em *EdgeMap[E]) f(u, v int, e E, backwards bool) bool {
	return em.fa(u, v, e, backwards)
}

// edgeMapSparseParallel parallelizes across CPU workers while preserving
// input order: it splits the vertices list into chunks, each worker
// processes its range in order, collecting targets per u in adjacency order.
func (em *EdgeMap[E]) edgeMapSparse(vertices []int) []int {
	n := len(vertices)

    // Determine number of parallel workers (<= len(vertices))
	// Choose a level of parallelism equal to the number of logical CPU cores available
    workers := runtime.NumCPU()
    if n < workers {
        workers = n
    }
	// Calculate chunkSize so that the vertices slice is evenly divided among workers, rounding up to cover all elements
    chunkSize := (n + workers - 1) / workers

	// Step 1: process each chunk in parallel

    // results[w] holds the flattened matches for the w-th chunk
    results := make([][]int, workers)
    var wg sync.WaitGroup
	// Start a loop over each worker index w to launch one goroutine per chunk
    for w := 0; w < workers; w++ {
        // Compute the starting index start for the w-th chunk in the vertices slice
		start := w * chunkSize
        // If start is out of bounds (no vertices left), skip launching a goroutine for this worker
		if start >= n {
            continue
        }
        end := start + chunkSize
        if end > n {
            end = n
        }
        wg.Add(1)
		// Launch a goroutine capturing the worker index w and its slice bounds s, e
        go func(w, s, e int) {
            defer wg.Done()
			// Within each worker, declare localFlat to collect this chunk’s matching targets in order
            var localFlat []int
            // Process vertices[s:e] in original input order
            for i := s; i < e; i++ {
                u := vertices[i]
                // Traverse G[u] in deterministic adjacency order
                for _, edge := range em.G[u] {
                    v := em.get(edge)
                    if em.cond(v) && em.f(u, v, edge, false) {
                        localFlat = append(localFlat, v)
                    }
                }
            }
            results[w] = localFlat
        }(w, start, end)
    }
    wg.Wait()

    // Flatten chunked results in worker index order to preserve global ordering
    total := 0
    for _, chunk := range results {
        total += len(chunk)
    }
    res := make([]int, 0, total)
    for _, chunk := range results {
        res = append(res, chunk...)
    }
    return res
}

// edgeMapDense implements the dense edge_map.
// It scans every vertex in the graph. For each vertex v for which cond(v)
// is true, it checks all its in-edges in GT[v]. Depending on the
// exitEarly flag, it either stops at the first matching edge or
// aggregates over all edges using a logical OR.
func (em *EdgeMap[E]) edgeMapDense(vertices []bool, exitEarly bool) []bool {
	result := make([]bool, em.n)
    var wg sync.WaitGroup

    for v := 0; v < em.n; v++ {
        wg.Add(1)
        go func(v int) {
            defer wg.Done()
            if !em.cond(v) {
                result[v] = false
                return
            }
            if exitEarly {
                found := false
                for _, e := range em.GT[v] {
                    u := em.get(e)
                    if vertices[u] && em.f(u, v, e, true) {
                        found = true
                        break
                    }
                }
                result[v] = found
            } else {
                resFlag := false
                for _, e := range em.GT[v] {
                    u := em.get(e)
                    if vertices[u] && em.f(u, v, e, true) {
                        resFlag = true
                        break
                    }
                }
                result[v] = resFlag
            }
        }(v)
    }
    wg.Wait()
    return result
}

// countTrue is a helper that counts how many elements in a bool slice are true.
func countTrue(b []bool) int {
	n := len(b)
    workers := runtime.NumCPU()
    chunk := (n + workers - 1) / workers
    var wg sync.WaitGroup
    ch := make(chan int, workers)

    for i := 0; i < n; i += chunk {
        end := i + chunk
        if end > n {
            end = n
        }
        wg.Add(1)
        go func(slice []bool) {
            defer wg.Done()
            cnt := 0
            for _, v := range slice {
                if v {
                    cnt++
                }
            }
            ch <- cnt
        }(b[i:end])
    }
    wg.Wait()
    close(ch)

    total := 0
    for c := range ch {
        total += c
    }
    return total
}

// Run is analogous to the overloaded operator() in the C++ code.
// It decides whether to use the sparse or dense method based on the size
// of the input vertex subset and then returns a new VertexSubset as result.
func (em *EdgeMap[E]) Run(vs VertexSubset, exitEarly bool) VertexSubset {
	// parallel count of active vertices
    var activeCount int
    if vs.isSparse {
        activeCount = len(vs.sparse)
    } else {
        activeCount = countTrue(vs.dense)
    }

    if vs.isSparse {
        // parallel compute incident edges count
        var wg sync.WaitGroup
        ch := make(chan int, len(vs.sparse))
        for _, v := range vs.sparse {
            wg.Add(1)
            go func(v int) {
                defer wg.Done()
                ch <- len(em.G[v])
            }(v)
        }
        go func() {
            wg.Wait()
            close(ch)
        }()
        d := 0
        for cnt := range ch {
            d += cnt
        }
        if (activeCount + d) > em.m/10 {
            dVertices := make([]bool, em.n)
            for _, i := range vs.sparse {
                dVertices[i] = true
            }
            newDense := em.edgeMapDense(dVertices, exitEarly)
            return NewDense(newDense)
        }
        newSparse := em.edgeMapSparse(vs.sparse)
        return NewSparse(newSparse)
    } else {
        if activeCount > em.n/20 {
            newDense := em.edgeMapDense(vs.dense, exitEarly)
            return NewDense(newDense)
        }
        seq := vs.ToSeq()
        newSparse := em.edgeMapSparse(seq)
        return NewSparse(newSparse)
    }
}

