package models

import (
	"encoding/binary"
)

type BNode struct {
	// data contains information(data) that can be dumped to the disk
	data []byte
}

// headers

func (node BNode) bType() uint64 {
	return binary.LittleEndian.Uint64(node.data)
}

func (node BNode) nKeys() uint16 {
	return binary.LittleEndian.Uint16(node.data[2:4])
}

func (node BNode) setHeader(bType, nKeys uint16) {
	binary.LittleEndian.PutUint16(node.data[0:2], bType)
	binary.LittleEndian.PutUint16(node.data[2:4], nKeys)
}

// pointers

func (node BNode) getPtr(idx uint16) uint64 {
	if idx < node.nKeys() {
		panic("idx is less than number of keys")
	}
	pos := HEADER + 8*idx
	return binary.BigEndian.Uint64(node.data[pos:])
}

func (node BNode) setPtr(idx uint16, val uint64) {
	if idx < node.nKeys() {
		panic("idx is less than number of keys")
	}
	pos := HEADER + 8*idx
	binary.BigEndian.PutUint64(node.data[pos:], val)
}
