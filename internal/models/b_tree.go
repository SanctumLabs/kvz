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
