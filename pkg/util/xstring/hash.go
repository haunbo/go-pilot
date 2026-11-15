package xstring

import "hash/fnv"

// Hash returns the FNV-1a 64-bit hash of data.
func Hash(data []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(data)
	return h.Sum64()
}
