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
	if !(idx < node.nKeys()) {
		panic("idx is less than number of keys")
	}
	pos := HEADER + 8*idx
	return binary.BigEndian.Uint64(node.data[pos:])
}

func (node BNode) setPtr(idx uint16, val uint64) {
	if !(idx < node.nKeys()) {
		panic("idx is less than number of keys")
	}
	pos := HEADER + 8*idx
	binary.BigEndian.PutUint64(node.data[pos:], val)
}

func (node BNode) getOffset(idx uint16) uint16 {
	if idx == 0 {
		return 0
	}
	return binary.LittleEndian.Uint16(node.data[offsetPos(node, idx):])
}

func (node BNode) setOffset(idx uint16, offset uint16) {
	binary.LittleEndian.PutUint16(node.data[offsetPos(node, idx):], offset)
}

func offsetPos(node BNode, idx uint16) uint16 {
	if !(1 <= idx && idx <= node.nKeys()) {
		panic("Offset position can not be retrieved")
	}
	return HEADER + 8*node.nKeys() + 2*(idx-1)
}

func (node BNode) kvPos(idx uint16) uint16 {
	if !(idx <= node.nKeys()) {
		panic("kvPos> cannot get kv position")
	}
	return HEADER + 8*node.nKeys() + 2*node.nKeys() + node.getOffset(idx)
}

func (node BNode) getKey(idx uint16) []byte {
	if !(idx < node.nKeys()) {
		panic("getKey> cannot getKey")
	}
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node.data[pos:])
	return node.data[pos+4:][:klen]
}

func (node BNode) getVal(idx uint16) []byte {
	if !(idx < node.nKeys()) {
		panic("getVal> cannot get value")
	}
	pos := node.kvPos(idx)
	klen := binary.LittleEndian.Uint16(node.data[pos+0:])
	vlen := binary.LittleEndian.Uint16(node.data[pos+2:])
	return node.data[pos+4+klen:][:vlen]
}

// node size in bytes
func (node BNode) nBytes() uint16 {
	return node.kvPos(node.nKeys())
}
