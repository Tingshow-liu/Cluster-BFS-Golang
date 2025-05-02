### 1. Compile C++ wrapper to import Cgo to GO
```
g++ -std=c++17 \
    -I. \
    -Icwrapper \
    -Ivendor \
    -Ivendor/src \
    -c cwrapper/wrapper.cpp \
    -o cwrapper/wrapper.o
```
### 2. Compile to GO executable
```
go build
```
### 3. Run the test
Example:
```
./cluster_bfs_go -f data/skitter_sym.bin -t 1 -ns 5 -k 5 -r 2 -v
```
