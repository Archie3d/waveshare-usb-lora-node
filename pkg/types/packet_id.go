package types

import (
	"math/rand/v2"
	"slices"
)

type PacketIdGenerator struct {
	prev  []uint32
	index int
}

func NewPacketIdGenerator(n uint) *PacketIdGenerator {
	return &PacketIdGenerator{
		prev:  make([]uint32, n),
		index: 0,
	}
}

func (p *PacketIdGenerator) GetNext() uint32 {
	v := rand.Uint32()
	for !p.isUnique(v) {
		v = rand.Uint32()
	}

	p.prev[p.index] = v
	p.index = (p.index + 1) % len(p.prev)

	return v
}

func (p *PacketIdGenerator) isUnique(value uint32) bool {
	return !slices.Contains(p.prev, value)
}
