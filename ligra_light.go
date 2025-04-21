package main

// analogue to Parlay's parallel loops
import (
	"cluster_bfs_go/parlay_go"
	"sync"
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
		old := vs.sparse
		total := len(old) + len(V)
		// Allocate a brand-new slice, like parlay::append returns a new sequence
		combined := make([]int, total)

		// parallel copy old data into combined[:len(old)]
		parlay_go.Append(old, combined[:len(old)])
		// parallel copy new vertices into combined[len(old):]
		parlay_go.Append(V, combined[len(old):])
		vs.sparse = combined
	} else {
		// Mimic Ligra in C++: a plain (sequential) for-loop is V is dense
		for _, v := range V {
			vs.dense[v] = true
		}
	}
	vs.n += len(V) // Same as "n += V.size();"
}

// ToSeq returns a unified slice of vertices regardless
// of the current representation.
func (vs *VertexSubset) ToSeq() []int {
	if vs.isSparse {
		return vs.sparse
	}
	return parlay_go.PackIndex(vs.dense)
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
	n    int                                      // number of vertices in the graph
	m    int                                      // total number of edges in the graph
	fa   func(u, v int, e E, backwards bool) bool // processing function for each edge
	get  func(e E) int                            // extracts a vertex from an edge; identity by default
	cond func(v int) bool                         // condition to test if vertex v qualifies
	G    [][]E                                    // forward graph (adjacency list for each vertex)
	GT   [][]E                                    // transposed graph for backward traversal
}

// NewEdgeMap constructs a new EdgeMap. It computes n (number of vertices)
// and m (total number of edges) from the input graph G.
func NewEdgeMap[E any](G, GT [][]E, fa func(u, v int, e E, backwards bool) bool,
	cond func(v int) bool, get func(e E) int) *EdgeMap[E] {
	n := len(G)
	m := 0
	for _, edges := range G {
		m += len(edges)
	}
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

// edgeMapSparse implements the sparse edge_map.
// It iterates over the sparse list of vertices and for each vertex u,
// it iterates over its outgoing edges in G[u]. For every edge e,
// it extracts the target vertex v = get(e), tests cond(v),
// and if em.f(u,v,e,false) returns true, it includes v in the result.
func (em *EdgeMap[E]) edgeMapSparse(vertices []int) []int {
	var res []int
	for _, u := range vertices {
		for _, e := range em.G[u] {
			v := em.get(e)
			if em.cond(v) && em.f(u, v, e, false) {
				res = append(res, v)
			}
		}
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
	for v := 0; v < em.n; v++ {
		if em.cond(v) {
			if exitEarly {
				found := false
				for _, e := range em.GT[v] {
					// Although the original C++ code includes a redundant cond check here,
					// we continue to process the edge.
					u := em.get(e)
					if vertices[u] && em.f(u, v, e, true) && !found {
						found = true
					}
				}
				result[v] = found
			} else {
				res := false
				for _, e := range em.GT[v] {
					u := em.get(e)
					res = res || (vertices[u] && em.f(u, v, e, true))
				}
				result[v] = res
			}
		} else {
			result[v] = false
		}
	}
	return result
}

// countTrue is a helper that counts how many elements in a bool slice are true.
func countTrue(b []bool) int {
	count := 0
	for _, v := range b {
		if v {
			count++
		}
	}
	return count
}

// Run is analogous to the overloaded operator() in the C++ code.
// It decides whether to use the sparse or dense method based on the size
// of the input vertex subset and then returns a new VertexSubset as result.
func (em *EdgeMap[E]) Run(vs VertexSubset, exitEarly bool) VertexSubset {
	var l int
	if vs.isSparse {
		l = vs.Size()
	} else {
		l = countTrue(vs.dense)
	}
	if vs.isSparse {
		// Compute the total number of edges incident on the sparse subset.
		d := 0
		for _, v := range vs.sparse {
			d += len(em.G[v])
		}
		// If the combined cost is greater than m/10, switch to dense.
		if (l + d) > em.m/10 {
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
		if l > em.n/20 {
			newDense := em.edgeMapDense(vs.dense, exitEarly)
			return NewDense(newDense)
		}
		// If the dense set is too small, convert to a sparse list.
		newSparse := em.edgeMapSparse(vs.ToSeq())
		return NewSparse(newSparse)
	}
}
