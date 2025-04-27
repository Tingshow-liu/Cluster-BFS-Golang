#!/usr/bin/env python3
import struct, sys, pathlib

def read_txt(path):
    edges = {}
    with open(path) as f:
        for line in f:
            if not line.strip():
                continue
            nums = list(map(int, line.split()))
            src, neigh = nums[0], nums[1:]
            edges[src] = neigh
    n = max(edges) + 1                      # assumes 0-based indices
    edges_list = [edges.get(i, []) for i in range(n)]
    return n, edges_list

def write_csr(n, edges_list, out_path):
    m = sum(len(neigh) for neigh in edges_list)
    offsets = [0]
    for neigh in edges_list:
        offsets.append(offsets[-1] + len(neigh))
    flat_edges = [u for neigh in edges_list for u in neigh]
    sizes = (n + 1) * 8 + m * 4 + 3 * 8

    with open(out_path, "wb") as f:
        pack = lambda fmt, x: f.write(struct.pack(fmt, x))
        for x in (n, m, sizes):         # header
            pack("<Q", x)
        for o in offsets:               # offsets
            pack("<Q", o)
        for e in flat_edges:            # edges
            pack("<I", e)
    print(f"Wrote {out_path}: n={n}, m={m}, sizes={sizes}")

if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("Usage: txt2csr.py graph.txt graph.bin")
    n, edges_list = read_txt(sys.argv[1])
    write_csr(n, edges_list, sys.argv[2])