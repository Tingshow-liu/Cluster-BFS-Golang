#ifndef WRAPPER_H
#define WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

// Call once before any BFS runs.  CSR arrays must stay alive until FreeLigraGraph.
void InitLigraGraph(
    const int* offsG,  int n_offG,
    const int* edgesG, int n_edgesG,
    const int* offsGT, int n_offGT,
    const int* edgesGT, int n_edgesGT
);

// Run Ligra::BFS on the prebuilt graph; writes into dist_out[0..n-1].
void RunLigraBFS_CSR(
    int start,
    unsigned long* dist_out    // length = n_offG-1
);

// Free the graphs you built in InitLigraGraph.
void FreeLigraGraph();

#ifdef __cplusplus
}
#endif
#endif