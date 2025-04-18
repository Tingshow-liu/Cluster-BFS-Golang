package bitutils

import "sync/atomic"

// FetchOrUint64 performs an atomic OR on *addr with mask
// addr: pointer to the
// mask: the bitmask to OR into addr
func FetchOr(addr *uint64, mask uint64) { // Capital starting letter: this function is Public to other files/ packages
	for {
		old := atomic.LoadUint64(addr) // Read the current value
		newVal := old | mask           // Compute the new value (OR)
		// Attempt to atomically replace the old value with the new value, retry (the for block) if someone else changed it in the meantime
		if atomic.CompareAndSwapUint64(addr, old, newVal) {
			return
		}
	}
}
