#include <atomic>
#include <utility>
#include <unordered_map>

#include <parlay/primitives.h>
#include <parlay/sequence.h>
#include <parlay/io.h>

#include "ligra_light.h"
#include "BFS_ligra.h"

/*
n is the number of vertices in the entire graph.
Cluster BFS efficiently finds the distances between all seeds and the vertices that are in the same cluster with those seeds — 
where 'same cluster' means reachable within R hops — by scanning the entire graph once.
*/

/*
Each vertex is processed in parallel (with threads/ goroutine) to find out its distances from all seeds that are within the same cluster (<= R) 
with bit-level parallelism, where each vertex's bitmask tracks which seeds have reached it and at what relative distance. Therefore, each goroutine 
will manage its corresponding S[v] and D[v] in parallel.
*/

// function template: announces that distance, grach, vertex, and label are all "names of different data types", ex: type distance, type vertex, vice versa
template <typename vertex, typename graph, typename distance, typename label, int R>
// vertices: An array of initial frontiers (seeds) of the cluster
/*
S: a nested array of length n (number of total vertices in the cluster); each inner array is of length R (cluster diameter + 1); every element 
of the inner arrays is a 64‑bit bitmask that can hold the up‑to‑64 seeds (uint_64). So S[v][r], where "r" is "relative round (after vertex v is 
reached by any seed vertex, it takes "r" round for other seed vertices to reach v)", represents what seeds reach vertex v after r rounds (at most R rounds)
*/
// D: For each vertex v, the shortest distance (in BFS hops) from any of the seed vertices (in vertices) in the cluster to v
void cluster_BFS(parlay::sequence<vertex>& vertices, const graph& G, parlay::sequence<std::array<label,R>>& S, parlay::sequence<distance>& D){
  parlay::internal::timer t("cluster_BFS",false);
  size_t n = G.size();  // Number of total vertices in graph G
  const distance INF=(1<<sizeof(distance)*8-1)-1;  // distance: an "intiger" between "0 and 255" (if it's uint_8)
  parlay::sequence<std::atomic<label>> S0(n);  // S0: a sequence of bitmasks that tracks which seed(s) have already visited vertex v in previous rounds
  parlay::sequence<std::atomic<label>> S1(n);  // S1: for each vertex v, S1[v] is a bitmask of seeds that have reached or are reaching v during this current round. It is updated in parallel as edges u → v are processed.
  parlay::sequence<std::atomic<distance>> distances(n);  // distances[v]: used to ensure that each node is only updated once per round to avoid race condition between multiple threads
  distance round = 0;
  
  // parlay::parallel_for: a function that runs a for loop in parallel in C++
  // Initialize all the variables needed to start the bit-parallel BFS, for every vertex i in the graph
  parlay::parallel_for(0,n, [&](vertex i){
    S0[i]=0;S1[i]=0;D[i]=INF;distances[i]=INF;
    for (int j = 0;j<R; j++){S[i][j]=0;}
  });

  // Initialize the seed vertices (i.e., the starting points of BFS) by:
  // 1. Adding each seed to a new sequence called seeds
  // 2. Setting their bits in S1[v] using S1[v] = 1ul << i, meaning thes seeds are "reached by themselves in this initialization round"”
  parlay::sequence<vertex> seeds;
  for (size_t i = 0; i<vertices.size(); i++){
    vertex v = vertices[i];
    if (i != 0 && v==vertices[0]){break;}
    S1[v] = 1ul <<i;
    seeds.push_back(v);
  }
  t.next("init");
  // printf("seeds: ");
  // for (auto v:seeds){
  //   printf("%d ", v);
  // }
  // printf("\n");
  
  // edge_f: a bit-level parallel function, run by many threads (thread-level parallelism)
  // edge_f (EdgeFunc) is used to decide whether this vertex v should be added to the frontier for this round and update S1[v]
  auto edge_f = [&] (vertex u, vertex v) -> bool {
    bool success= false;
    // u tries to tell v what seeds visited u, so v can be reached by these seeds that visited u
    label u_visited = S0[u].load();  // seeds that reached u in earlier rounds
    label v_visited = S1[v].load();  // seeds already marked as reaching v (in this round)
    if ((u_visited | v_visited) != v_visited) {
      // some seeds that reached u haven't reached v yet
      S1[v].fetch_or(u_visited);  // let v inherit those seed visits from u
      distance old_d = distances[v].load();
      // compare_exchange_strong(): a thread-safe atomic operation
      // distances[v].compare_exchange_strong(expected_val, new_val): 
      // 1. if distances[v] == expected_val, atomically updates distances[v] to new_val
      // 2. Return true if the update happened; return false if any other thread changed it before this thread makes the change
      if(old_d != round && distances[v].compare_exchange_strong(old_d, round))
        success=true;
    };
    return success;
  };

  // frontier_f: runs after a vertex v that has been updated this round, and updates its records
  // frontier_f (FrontierFunc) is used to update D, S, and S0
  auto frontier_f = [&](vertex v){
    // S1[v] = all seeds that tried to reach v in this round
    // S0[v] = all seeds that had already reached v before this round
    // So difference = new seeds that just reached v this round
    label difference = S1[v].load() &~ S0[v].load();
    // If this is the first time v has been visited, set its BFS round (D[v])
    if (D[v]==INF){D[v]=round;}
    // The key bit-parallelism line:
    // S[v][r] stores which seeds reached v at relative round r;
    // -> round - D[v] (current round in BFS - the round when vertex v is first visited) gives that relative round number
    S[v][round-D[v]]=difference;
    // Update S0[v] to include the new seeds — so in the next round, these won’t be counted again
    S0[v]|=difference;
  };

  // cond_f: decides which vertices should be considered for updates in the current BFS round
  // Returns true if:
  // Vertices that haven’t been visited yet (D[v] == INF)
  // Or were recently visited and still within the radius R
  /*
  In bit-parallel BFS, need to revisit nodes after they’re discovered, up to R rounds, 
  to update their bitmasks, which is not needed in regular BFS!
  */
  auto cond_f = [&] (vertex v) {return D[v]==INF || round-D[v]<R;};
  // frontier_map: sets up the actual parallel BFS traversal
  // This is Ligra's thread-level parallelism engine, but you’ve now customized it with bit-level logic inside edge_f
  auto frontier_map = ligra::edge_map(G, G, edge_f, cond_f);
  
  // frontier: initializes the first frontiers of the BFS (cluster)
  auto frontier = ligra::vertex_subset<vertex>();
  frontier.add_vertices(seeds);
  t.next("head");

  long total = 0;
  vertex v = 99;
  // Inner loop for BFS within a single cluster.
  while (frontier.size() > 0) {
    frontier.apply(frontier_f);
    round++;
    long m = frontier.size();
    total += m;
    frontier = frontier_map(frontier, false);
    // t.next("map");
    t.next("update");
  }
}


template <typename vertex,typename graph, typename distance, typename label, int R>
void verify_CBFS(parlay::sequence<vertex>& vertices, const graph& G, parlay::sequence<std::array<label,R>>& S, parlay::sequence<distance>& D){
  const distance INF=(1<<sizeof(distance)*8-1)-1;
  size_t n = G.size();
  parlay::sequence<distance> answer(n);
  for (size_t j = 0; j<vertices.size(); j++){
    if (j!=0 && vertices[j]==vertices[0]){break;}
    BFS(vertices[j], G, G, answer);
    for (size_t v = 0; v<n; v++){
      distance d_true = answer[v];
      distance d_query=D[v];
      if (d_true == INF){continue;}
      label sum = 0;
      bool changed=false;
      for (size_t r = 0;r<R; r++){
        sum|=S[v][r];
        if (sum&(1ul<<j)){
          d_query += r;
          changed=true;
          break;
        }
      }
      
      if (changed && d_query!=d_true){
        printf("source: %d, vertex_id %d, target: %d, d_true: %d, d_query: %d, D: %d\n",vertices[j], j,v,d_true, d_query, D[v]);
        for (int i = 0; i<R; i++){
          printf("S[%d]: %lu\n", i, S[v][i]);
        }
        return;
      }else if (!changed && d_true-d_query>((R+1)/2)*2){
        printf("source: %d, vertex_id %d, target: %d, d_true: %u, d_query: %u out of range\n", vertices[j],j,v,d_true, d_query);
        return;
      }
    }
  }
  printf("PASS\n");
}