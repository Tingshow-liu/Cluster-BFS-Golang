package graphutils

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Convert the first several lines of bin data to a human readable format
func peek(path string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Read the first 10 unit64 words:
	buf := make([]byte, 8*5)
	if _, err := io.ReadFull(f, buf); err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		x := binary.LittleEndian.Uint64(buf[i*8 : i*8+8])
		names := []string{"n", "m", "sizes", "offset[0]", "offset[1]"}
		fmt.Printf("%-10s = %d\n", names[i], x)
	}
}
