package models

const (
	// internal nodes without values
	BNODE_NODE = 1

	// Leaf nodes with values
	BNODE_LEAF = 2

	HEADER = 4

	BTREE_PAGE_SIZE    = 4096
	BTREE_MAX_KEY_SIZE = 1000
	BTREE_MAX_VAL_SIZE = 3000
)
