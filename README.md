## Cluster-BFS in Golang

### Download datasets
```
python3 download.py
```

### Compile C++ wrapper to use Cgo for Cluster-BFS (in Golang) output verification
```
g++ -std=c++17 \
    -I. \
    -Icwrapper \
    -Ivendor \
    -Ivendor/src \
    -c cwrapper/wrapper.cpp \
    -o cwrapper/wrapper.o
```

### Compile to Go executable
```
go build
```

### Run the test
The program accepts the following flags to configure its behavior:
| Flag      | Type    | Description |
|-----------|---------|-------------|
| `-f`      | string  | **(Required)** Path to the data file (ex: data/graphs/com-youtube_sym.bin) to be loaded. |
| `-t`      | int     | Number of iterations to run the test. Default: `3`. |
| `-ns`     | int     | Number of seed batches. Default: `10`. |
| `-k`      | int     | Number of seeds per batch. Default: `64`. |
| `-r`      | int     | BFS radius used for result verification. Default: `2`. |
| `-v`      | bool    | Whether to verify with Ligra BFS (`true` to enable). Default: `false`. |
| `-seq`    | bool    | If `true`, run Sequential BFS; if `false`, run ClusterBFS. Default: `false`. |
| `-c`      | int     | Number of CPU cores to use (`GOMAXPROCS`). Default: `20`. |

Example commands:
```
./cluster_bfs_go -f data/graphs/Epinions1_sym.bin
./cluster_bfs_go -f data/graphs/Epinions1_sym.bin -t 1 -ns 5 -k 5 -r 2 -v -c 4
```
