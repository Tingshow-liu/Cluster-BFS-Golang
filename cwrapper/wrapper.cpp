#include "wrapper.h"
#include <vendor/parlay/sequence.h>
#include <vendor/parlay/primitives.h>
#include <vendor/src/ligra_light.h>
#include <vendor/src/BFS_ligra.h>
#include <array>
#include <atomic>
#include <vector>

// Forward declare your Ligra BFS template:
template <typename V, typename Graph, typename Dist>
void BFS(V start, const Graph& G, const Graph& GT, parlay::sequence<Dist>& dist);

// Build a Ligra graph from CSR once:
static parlay::sequence<parlay::sequence<int>> *Gptr = nullptr, *GTptr = nullptr;
static parlay::sequence<parlay::sequence<int>> build_graph(
    const int* offs, int n_off, const int* edges) {

  size_t n = size_t(n_off - 1);
  // // — log how many vertices and the first/last offset index —
  // std::cerr
  // << "[build_graph] n_vertices=" << n
  // << ", offs[0]=" << offs[0]
  // << ", offs[" << n << "]=" << offs[n]
  // << "\n";
  parlay::sequence<parlay::sequence<int>> G(n);
  for (size_t i = 0; i < n; i++) {
    int a = offs[i], b = offs[i+1];
    // you can even log each range if you suspect corruption here:
    // std::cerr << "  vertex " << i << ": edges[" << a << "," << b << ")\n";
    G[i] = parlay::sequence<int>(edges + a, edges + b);
  }
  return G;
}

void InitLigraGraph(
    const int* offsG,  int n_offG,
    const int* edgesG, int n_edgesG,
    const int* offsGT, int n_offGT,
    const int* edgesGT, int n_edgesGT
) {
  // // — log the raw CSR sizes passed from Go —
  // std::cerr
  // << "[InitLigraGraph] G:  n_off=" << n_offG
  // << ", n_edges=" << n_edgesG
  // << "   |   GT: n_off=" << n_offGT
  // << ", n_edges=" << n_edgesGT
  // << "\n";
  auto G  = build_graph(offsG,  n_offG,  edgesG);
  auto GT = build_graph(offsGT, n_offGT, edgesGT);
  Gptr  = new decltype(G)(std::move(G));
  GTptr = new decltype(GT)(std::move(GT));
}

void RunLigraBFS_CSR(int start, unsigned long* dist_out) {
  using vertex   = int;
  using distance = unsigned long;
  size_t n = Gptr->size();

  parlay::sequence<distance> dist(n);
  BFS<vertex, decltype(*Gptr), distance>(start, *Gptr, *GTptr, dist);

  for (size_t i = 0; i < n; i++) {
    dist_out[i] = dist[i];
  }
}

void FreeLigraGraph() {
  delete Gptr;
  delete GTptr;
  Gptr = nullptr;
  GTptr = nullptr;
}