package models

import (
	"bytes"
	"encoding/binary"
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

func leafInsert(new BNode, old BNode, idx uint16, key []byte, val []byte) {
	new.setHeader(BNODE_LEAF, old.nKeys()+1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendKV(new, idx, 0, key, val)
	nodeAppendRange(new, old, idx+1, idx, old.nKeys()-idx)
}

// nodeAppendRange copies multiple KVs into the position
func nodeAppendRange(new BNode, old BNode, dstNew uint16, srcOld uint16, n uint16) {
	if !(srcOld+n <= old.nKeys()) {
		panic("srcOld +n > old.nKeys()")
	}

	if !(dstNew+n <= new.nKeys()) {
		panic("dstNew+n > new.nKeys()")
	}

	if n == 0 {
		return
	}

	// pointers
	for i := uint16(0); i < n; i++ {
		new.setPtr(dstNew+i, old.getPtr(srcOld+i))
	}

	//offsets
	dstBegin := new.getOffset(dstNew)
	srcBegin := old.getOffset(srcOld)

	// Note: the range is [1,n]
	for i := uint16(1); i <= n; i++ {
		offset := dstBegin + old.getOffset(srcOld+i) - srcBegin
		new.setOffset(dstNew+i, offset)
	}

	// KVs
	begin := old.kvPos(srcOld)
	end := old.kvPos(srcOld + n)
	copy(new.data[new.kvPos(dstNew):], old.data[begin:end])
}

// nodeAppendKV copies a KV pair to the new node
func nodeAppendKV(new BNode, idx uint16, ptr uint64, key []byte, val []byte) {
	// pointers
	new.setPtr(idx, ptr)

	// KVs
	pos := new.kvPos(idx)
	binary.LittleEndian.PutUint16(new.data[pos+0:], uint16(len(key)))
	binary.LittleEndian.PutUint16(new.data[pos+2:], uint16(len(val)))

	copy(new.data[pos+4:], key)
	copy(new.data[pos+4+uint16(len(key)):], val)

	// the offset of the next key
	new.setOffset(idx+1, new.getOffset(idx)+4+uint16((len(key)+len(val))))
}

// treeInsert inserts a KV into a node, the result might be split into 2 nodes.
// the caller is responsible for deallocating the input node
// and splitting and allocating result nodes.
func treeInsert(tree *BTree, node BNode, key []byte, val []byte) BNode {
	// the result node. It's allowed to be bigger than 1 page and will be split if so
	new := BNode{data: make([]byte, 2*BTREE_PAGE_SIZE)}

	// where to insert the key?
	idx := NodeLookupLE(node, key)

	// act depending on the node type
	switch node.bType() {
	case BNODE_LEAF:
		// leaf, node.getKey(idx) <= key
		if bytes.Equal(key, node.getKey(idx)) {
			// found the key, update it.
			leafInsert(new, node, idx, key, val)
		} else {
			// insert it after the position
			leafInsert(new, node, idx+1, key, val)
		}
	case BNODE_NODE:
		// internal node, insert it to a kid node
		nodeInsert(tree, new, node, idx, key, val)
	default:
		panic("unknown node type")
	}

	return new
}

// nodeInsert handles insertion of internal nodes into the tree
func nodeInsert(tree *BTree, new BNode, node BNode, idx uint16, key []byte, val []byte) {
	// get and deallocate the kid node
	kptr := node.getPtr(idx)
	knode := tree.get(kptr)
	tree.del(kptr)

	// recursive insertion to the kid node
	knode = treeInsert(tree, knode, key, val)

	// split the result
	nsplit, splited := nodeSplit3(node)

	// update the kid links
	nodeReplaceKidN(tree, new, node, idx, splited[:nsplit]...)
}

// nodeSplit3 splits a bigger-than-allowed node int 2. The 2nd node always fits on a page
func nodeSplit2(left, right, old BNode) {
}

// nodeSplit3 splits a node if it's too big. The results are 1~3 nodes
func nodeSplit3(old BNode) (uint16, [3]BNode) {
	if old.nBytes() <= BTREE_PAGE_SIZE {
		old.data = old.data[:BTREE_PAGE_SIZE]
		return 1, [3]BNode{old}
	}

	left := BNode{make([]byte, 2*BTREE_PAGE_SIZE)} // might be split later
	right := BNode{make([]byte, BTREE_PAGE_SIZE)}
	nodeSplit2(left, right, old)

	if left.nBytes() <= BTREE_PAGE_SIZE {
		left.data = left.data[:BTREE_PAGE_SIZE]
		return 2, [3]BNode{left, right}
	}

	// the left node is still too loarge
	leftLeft := BNode{make([]byte, BTREE_PAGE_SIZE)}
	middle := BNode{make([]byte, BTREE_PAGE_SIZE)}
	nodeSplit2(leftLeft, middle, left)
	if !(leftLeft.nBytes() <= BTREE_PAGE_SIZE) {
		panic("leftLeft.nBytes() > BTREE_PAGE_SIZE")
	}
	return 3, [3]BNode{leftLeft, middle, right}
}

// nodeReplaceKidN replaces a link with multiple links
func nodeReplaceKidN(tree *BTree, new BNode, old BNode, idx uint16, kids ...BNode) {
	// TODO: what value is inc?
	inc := uint16(1)
	new.setHeader(BNODE_NODE, old.nKeys()+inc-1)
	nodeAppendRange(new, old, 0, 0, idx)
	for i, node := range kids {
		nodeAppendKV(new, idx+uint16(i), tree.new(node), node.getKey(0), nil)
	}
	nodeAppendRange(new, old, idx+inc, idx+1, old.nKeys()-(idx+1))
}

// leafDelete removes a key from a leaf node
func leafDelete(new BNode, old BNode, idx uint16) {
	new.setHeader(BNODE_LEAF, old.nKeys()-1)
	nodeAppendRange(new, old, 0, 0, idx)
	nodeAppendRange(new, old, idx, idx+1, old.nKeys()-(idx+1))
}

// treeDelete deletes a key from a tree
func treeDelete(tree *BTree, node BNode, key []byte) BNode {
	// where to find the key
	idx := NodeLookupLE(node, key)

	// act depending on the node type
	switch node.bType() {
	case BNODE_LEAF:
		if !bytes.Equal(key, node.getKey(idx)) {
			return BNode{} //not found
		}
		// delete the key in the leaf
		new := BNode{data: make([]byte, BTREE_PAGE_SIZE)}
		leafDelete(new, node, idx)
		return new

	case BNODE_NODE:
		return nodeDelete(tree, node, idx, key)
	default:
		panic("bad node!")
	}
}

func nodeDelete(tree *BTree, node BNode, idx uint16, key []byte) BNode {
	// recurse into the kid
	kptr := node.getPtr(idx)
	updated := treeDelete(tree, tree.get(kptr), key)
	if len(updated.data) == 0 {
		return BNode{} // not found
	}
	tree.del(kptr)

	new := BNode{data: make([]byte, BTREE_PAGE_SIZE)}

	// check for merging
	mergeDir, sibling := shouldMerge(tree, node, idx, updated)
	switch {
	case mergeDir < 0: // left
		merged := BNode{data: make([]byte, BTREE_PAGE_SIZE)}
		nodeMerge(merged, sibling, updated)
		tree.del(node.getPtr(idx - 1))
		nodeReplace2Kid(new, node, idx-1, tree.new(merged), merged.getKey(0))
	case mergeDir > 0: // right
		merged := BNode{data: make([]byte, BTREE_PAGE_SIZE)}
		nodeMerge(merged, updated, sibling)
		tree.del(node.getPtr(idx + 1))
		nodeReplace2Kid(new, node, idx, tree.new(merged), merged.getKey(0))
	case mergeDir == 0:
		if !(updated.nKeys() > 0) {
			panic("number of keys not greater than 0")
		}
		nodeReplaceKidN(tree, new, node, idx, updated)
	}
	return new
}

// nodeMerge merges 2 nodes into 1
func nodeMerge(new, left, right BNode) {
	new.setHeader(uint16(left.bType()), left.nKeys()+right.nKeys())
	nodeAppendRange(new, left, 0, 0, left.nKeys())
	nodeAppendRange(new, right, left.nKeys(), 0, right.nKeys())
}

// shouldMerge returns the idx of the kid and the node. Checks if the updated kid should be merged with a sibling
// The conditions are:
// 1. The node is smaller than 1/4 of a page
// 2. Has a sibling and the merged result does not exceed one page
func shouldMerge(tree *BTree, node BNode, idx uint16, updated BNode) (int, BNode) {
	if updated.nBytes() > BTREE_PAGE_SIZE/4 {
		return 0, BNode{}
	}

	if idx > 0 {
		sibling := tree.get(node.getPtr(idx - 1))
		merged := sibling.nBytes() + updated.nBytes() - HEADER

		if merged <= BTREE_PAGE_SIZE {
			return -1, sibling
		}
	}

	if idx+1 < node.nKeys() {
		sibling := tree.get(node.getPtr(idx + 1))
		merged := sibling.nBytes() + updated.nBytes() - HEADER

		if merged <= BTREE_PAGE_SIZE {
			return 1, sibling
		}
	}

	return 0, BNode{}
}
