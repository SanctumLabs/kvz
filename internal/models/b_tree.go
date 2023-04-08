package models

type BTree struct {
	// pointer ( a nonzero page number)
	root uint64

	// callbacks for managing on-disk pages

	// dereference a pointer
	get func(uint64) BNode

	// allocate a new page
	new func(BNode) uint64

	// deallocate a page
	del func(uint64)
}

func (tree *BTree) Delete(key []byte) bool {
	if !(len(key) != 0) {
		panic("Key is not equal to 0")
	}

	if !(len(key) <= BTREE_MAX_KEY_SIZE) {
		panic("length of key is greater than MAX KEY SIZE")
	}

	if tree.root == 0 {
		return false
	}

	updated := treeDelete(tree, tree.get(tree.root), key)
	if len(updated.data) == 0 {
		return false //not found
	}

	tree.del(tree.root)
	if updated.bType() == BNODE_NODE && updated.nKeys() == 1 {
		// remove a level
		tree.root = updated.getPtr(0)
	} else {
		tree.root = tree.new(updated)
	}

	return true
}

// Insert adds a new key value pair to the BTree
// a new root node is added when the old root is split into multiple nodes
// and when inserting the first key, it creates the first leaf node as the root
// We insert an empty key into the tree when we create the first node. The empty key is the lowest possible key by sorting order,
// it makes the lookup function nodeLookupLE always successful, eliminating the case of failing to find a node that contains the input key.
func (tree *BTree) Insert(key []byte, val []byte) {
	if !(len(key) != 0) {
		panic("Key is 0")
	}

	if !(len(key) <= BTREE_MAX_KEY_SIZE) {
		panic("Key is greater than max key size")
	}

	if !(len(val) <= BTREE_MAX_VAL_SIZE) {
		panic("value is greater than max value size")
	}

	if tree.root == 0 {
		// create the first node
		root := BNode{data: make([]byte, BTREE_PAGE_SIZE)}
		root.setHeader(BNODE_LEAF, 2)
		// a dummy key, this makes the tree cover the whole key space
		// thus a lookup can always find a containing node
		nodeAppendKV(root, 0, 0, nil, nil)
		nodeAppendKV(root, 1, 0, key, val)
		tree.root = tree.new(root)
		return
	}

	node := tree.get(tree.root)
	tree.del(tree.root)

	node = treeInsert(tree, node, key, val)
	nSplit, splitted := nodeSplit3(node)

	if nSplit > 1 {
		// the root was split, add a new level
		root := BNode{data: make([]byte, BTREE_PAGE_SIZE)}
		root.setHeader(BNODE_NODE, nSplit)

		for i, kNode := range splitted[:nSplit] {
			ptr, key := tree.new(kNode), kNode.getKey(0)
			nodeAppendKV(root, uint16(i), ptr, key, nil)
		}
		tree.root = tree.new(root)
	} else {
		tree.root = tree.new(splitted[0])
	}
}
