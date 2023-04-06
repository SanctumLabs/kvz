package models

import (
	"bytes"
)

// nodeLookupLE returns the first kid node whose range intersects the key. (kid[i] <= key)
// the lookup works for both leaf nodes and internal nodes. Note that the first key is skipped
// for comparison, since it has already been compared from the parent node
func NodeLookupLE(node BNode, key []byte) uint16 {
	nkeys := node.nKeys()
	found := uint16(0)

	// the first key is a copy from the parent node, thus it's always less than or equal
	// to the key
	for i := uint16(1); i < nkeys; i++ {
		cmp := bytes.Compare(node.getKey(i), key)
		if cmp <= 0 {
			found = i
		}

		if cmp >= 0 {
			break
		}
	}

	return found
}
