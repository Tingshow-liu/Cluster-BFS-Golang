package main

import (
    "sync"
    "sync/atomic"
)

// ----------------------------------------------------
// VertexSubset: parallel-capable representation of a set
// of vertices in either "sparse" or "dense" form.
// ----------------------------------------------------

type VertexSubset struct {
    isSparse bool
    n        int
    sparse   []int
    dense    []bool
}

// Size returns the number of vertices in the subset.
func (vs *VertexSubset) Size() int {
    return vs.n
}

// NewEmptySparse creates a new, empty sparse VertexSubset.
func NewEmptySparse() VertexSubset {
    return VertexSubset{isSparse: true, n: 0, sparse: []int{}}
}

// NewSparse creates a VertexSubset from a sparse slice.
func NewSparse(vertices []int) VertexSubset {
    return VertexSubset{isSparse: true, n: len(vertices), sparse: vertices}
}

// NewSingle creates a VertexSubset containing a single vertex.
func NewSingle(v int) VertexSubset {
    return VertexSubset{isSparse: true, n: 1, sparse: []int{v}}
}

// NewDense creates a VertexSubset from a dense boolean slice.
func NewDense(dense []bool) VertexSubset {
    return VertexSubset{isSparse: false, n: countTrue(dense), dense: dense}
}

// AddVertices adds vertices in parallel when dense.
func (vs *VertexSubset) AddVertices(V []int) {
    if vs.isSparse {
        vs.sparse = append(vs.sparse, V...)
        vs.n = len(vs.sparse)
    } else {
        var wg sync.WaitGroup
        for _, v := range V {
            wg.Add(1)
            go func(u int) {
                defer wg.Done()
                vs.dense[u] = true
            }(v)
        }
        wg.Wait()
        vs.n = countTrue(vs.dense)
    }
}

// ToSeq returns a slice of active vertices in parallel.
func (vs *VertexSubset) ToSeq() []int {
    if vs.isSparse {
        return vs.sparse
    }
    var mu sync.Mutex
    seq := make([]int, 0)
    var wg sync.WaitGroup
    for i, active := range vs.dense {
        wg.Add(1)
        go func(idx int, ok bool) {
            defer wg.Done()
            if ok {
                mu.Lock()
                seq = append(seq, idx)
                mu.Unlock()
            }
        }(i, active)
    }
    wg.Wait()
    return seq
}

// Apply applies f in parallel over the subset.
func (vs *VertexSubset) Apply(f func(int)) {
    var wg sync.WaitGroup
    if vs.isSparse {
        for _, v := range vs.sparse {
            wg.Add(1)
            go func(u int) {
                defer wg.Done()
                f(u)
            }(v)
        }
    } else {
        for i, active := range vs.dense {
            if active {
                wg.Add(1)
                go func(u int) {
                    defer wg.Done()
                    f(u)
                }(i)
            }
        }
    }
    wg.Wait()
}

// ----------------------------------------------------
// EdgeMap: parallel-capable edge mapping structure.
// ----------------------------------------------------

type EdgeMap[E any] struct {
    n    int
    m    int64
    fa   func(u, v int, e E, backwards bool) bool
    get  func(e E) int
    cond func(v int) bool
    G    [][]E
    GT   [][]E
}

// NewEdgeMap constructs a new EdgeMap computing m in parallel.
func NewEdgeMap[E any](G, GT [][]E,
    fa func(u, v int, e E, backwards bool) bool,
    cond func(v int) bool,
    get func(e E) int) *EdgeMap[E] {
    n := len(G)
    var total int64
    var wg sync.WaitGroup
    wg.Add(n)
    for i, edges := range G {
        go func(cnt int) {
            defer wg.Done()
            atomic.AddInt64(&total, int64(cnt))
        }(len(edges))
    }
    wg.Wait()
    return &EdgeMap[E]{n: n, m: total, fa: fa, get: get, cond: cond, G: G, GT: GT}
}

// f wrapper (always uses four arguments in Go).
func (em *EdgeMap[E]) f(u, v int, e E, backwards bool) bool {
    return em.fa(u, v, e, backwards)
}

// edgeMapSparse runs in parallel over the sparse vertex list.
func (em *EdgeMap[E]) edgeMapSparse(vertices []int) []int {
    var wg sync.WaitGroup
    var mu sync.Mutex
    res := make([]int, 0)
    wg.Add(len(vertices))
    for _, u := range vertices {
        go func(src int) {
            defer wg.Done()
            local := make([]int, 0)
            for _, e := range em.G[src] {
                v := em.get(e)
                if em.cond(v) && em.f(src, v, e, false) {
                    local = append(local, v)
                }
            }
            mu.Lock()
            res = append(res, local...)
            mu.Unlock()
        }(u)
    }
    wg.Wait()
    return res
}

// edgeMapDense runs in parallel over all vertices.
func (em *EdgeMap[E]) edgeMapDense(vertices []bool, exitEarly bool) []bool {
    result := make([]bool, em.n)
    var wg sync.WaitGroup
    wg.Add(em.n)
    for v := 0; v < em.n; v++ {
        go func(idx int) {
            defer wg.Done()
            if !em.cond(idx) {
                return
            }
            if exitEarly {
                found := false
                for _, e := range em.GT[idx] {
                    u := em.get(e)
                    if vertices[u] && em.f(u, idx, e, true) {
                        found = true
                        break
                    }
                }
                result[idx] = found
            } else {
                val := false
                for _, e := range em.GT[idx] {
                    u := em.get(e)
                    if vertices[u] && em.f(u, idx, e, true) {
                        val = true
                        // can break early for OR
                        break
                    }
                }
                result[idx] = val
            }
        }(v)
    }
    wg.Wait()
    return result
}

// Run decides between sparse/dense in parallel.
func (em *EdgeMap[E]) Run(vs VertexSubset, exitEarly bool) VertexSubset {
    var l int
    if vs.isSparse {
        l = vs.Size()
    } else {
        l = countTrue(vs.dense)
    }
    if vs.isSparse {
        // parallel compute total incident edges d
        var d int64
        var wg sync.WaitGroup
        wg.Add(len(vs.sparse))
        for _, u := range vs.sparse {
            go func(src int) {
                defer wg.Done()
                atomic.AddInt64(&d, int64(len(em.G[src])))
            }(u)
        }
        wg.Wait()
        if int64(l)+d > em.m/10 {
            dVertices := make([]bool, em.n)
            for _, i := range vs.sparse {
                dVertices[i] = true
            }
            return NewDense(em.edgeMapDense(dVertices, exitEarly))
        }
        return NewSparse(em.edgeMapSparse(vs.sparse))
    }
    if l > em.n/20 {
        return NewDense(em.edgeMapDense(vs.dense, exitEarly))
    }
    return NewSparse(em.edgeMapSparse(vs.ToSeq()))
}

// countTrue counts true entries sequentially.
func countTrue(b []bool) int {
    cnt := 0
    for _, v := range b {
        if v {
            cnt++
        }
    }
    return cnt
}
