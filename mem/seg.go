//-----------------------------------------------------------------------------
/*

Memory Segments

*/
//-----------------------------------------------------------------------------

package mem

import (
	"encoding/binary"
	"math"
	"strings"
)

//-----------------------------------------------------------------------------

// Exception is a bit mask of memory access exceptions.
type Exception uint

// Exception values.
const (
	ExAlign Exception = 1 << iota // misaligned read/write
	ExRead                        // can't read this memory
	ExWrite                       // can't write this memory
	ExEmpty                       // no memory at this address
	ExExec                        // can't read instructions from this memory
)

func (e Exception) String() string {
	s := make([]string, 0)
	if e&ExAlign != 0 {
		s = append(s, "align")
	}
	if e&ExRead != 0 {
		s = append(s, "read")
	}
	if e&ExWrite != 0 {
		s = append(s, "write")
	}
	if e&ExEmpty != 0 {
		s = append(s, "empty")
	}
	if e&ExExec != 0 {
		s = append(s, "exec")
	}
	return strings.Join(s, ",")
}

//-----------------------------------------------------------------------------

// Attribute is a bit mask of memory access attributes.
type Attribute uint

// Attribute values.
const (
	AttrR Attribute = 1 << iota // read
	AttrW                       // write
	AttrX                       // execute
)

// AttrRW = read/write
const AttrRW = AttrR | AttrW

// AttrRX = read/execute
const AttrRX = AttrR | AttrX

// AttrRWX = read/write/execute
const AttrRWX = AttrR | AttrW | AttrX

//-----------------------------------------------------------------------------
// memory access exceptions

func wrException(adr uint, attr Attribute, align uint) Exception {
	var ex Exception
	if attr&AttrW == 0 {
		ex |= ExWrite
	}
	if adr&(align-1) != 0 {
		ex |= ExAlign
	}
	return ex
}

func rdException(adr uint, attr Attribute, align uint) Exception {
	var ex Exception
	if attr&AttrR == 0 {
		ex |= ExRead
	}
	if adr&(align-1) != 0 {
		ex |= ExAlign
	}
	return ex
}

func rdInsException(adr uint, attr Attribute) Exception {
	// rv32c has mixed 32/16 bit instruction streams so
	// we allow 32-bit reads on 2 byte address boundaries.
	ex := rdException(adr, attr, 2)
	if attr&AttrX == 0 {
		ex |= ExExec
	}
	return ex
}

//-----------------------------------------------------------------------------

// Segment is an interface to a contiguous region of memory.
type Segment interface {
	RdIns(adr uint) (uint, Exception)
	Rd64(adr uint) (uint64, Exception)
	Rd32(adr uint) (uint32, Exception)
	Rd16(adr uint) (uint16, Exception)
	Rd8(adr uint) (uint8, Exception)
	Wr64(adr uint, val uint64) Exception
	Wr32(adr uint, val uint32) Exception
	Wr16(adr uint, val uint16) Exception
	Wr8(adr uint, val uint8) Exception
	In(adr, size uint) bool
}

//-----------------------------------------------------------------------------

// Chunk is a contiguous chunk of memory.
type Chunk struct {
	attr       Attribute // bitmask of attributes
	start, end uint      // address range
	mem        []uint8   // memory array
}

// NewChunk allocates and returns a memory chunk.
func NewChunk(start, size uint, attr Attribute) *Chunk {
	// allocate the memory and set it to all ones
	mem := make([]uint8, size)
	for i := range mem {
		mem[i] = 0xff
	}
	return &Chunk{
		attr:  attr,
		start: start,
		end:   start + size - 1,
		mem:   mem,
	}
}

// In returns true if the adr, size is entirely within the memory chunk.
func (m *Chunk) In(adr, size uint) bool {
	end := adr + size - 1
	return (adr >= m.start) && (end <= m.end)
}

// RdIns reads a 32-bit instruction from memory.
func (m *Chunk) RdIns(adr uint) (uint, Exception) {
	return uint(binary.LittleEndian.Uint32(m.mem[adr-m.start:])), rdInsException(adr, m.attr)
}

// Rd64 reads a 64-bit data value from memory.
func (m *Chunk) Rd64(adr uint) (uint64, Exception) {
	return binary.LittleEndian.Uint64(m.mem[adr-m.start:]), rdException(adr, m.attr, 8)
}

// Rd32 reads a 32-bit data value from memory.
func (m *Chunk) Rd32(adr uint) (uint32, Exception) {
	return binary.LittleEndian.Uint32(m.mem[adr-m.start:]), rdException(adr, m.attr, 4)
}

// Rd16 reads a 16-bit data value from memory.
func (m *Chunk) Rd16(adr uint) (uint16, Exception) {
	return binary.LittleEndian.Uint16(m.mem[adr-m.start:]), rdException(adr, m.attr, 2)
}

// Rd8 reads an 8-bit data value from memory.
func (m *Chunk) Rd8(adr uint) (uint8, Exception) {
	return m.mem[adr-m.start], rdException(adr, m.attr, 1)
}

// Wr64 writes a 64-bit data value to memory.
func (m *Chunk) Wr64(adr uint, val uint64) Exception {
	binary.LittleEndian.PutUint64(m.mem[adr-m.start:], val)
	return wrException(adr, m.attr, 8)
}

// Wr32 writes a 32-bit data value to memory.
func (m *Chunk) Wr32(adr uint, val uint32) Exception {
	binary.LittleEndian.PutUint32(m.mem[adr-m.start:], val)
	return wrException(adr, m.attr, 4)
}

// Wr16 writes a 16-bit data value to memory.
func (m *Chunk) Wr16(adr uint, val uint16) Exception {
	binary.LittleEndian.PutUint16(m.mem[adr-m.start:], val)
	return wrException(adr, m.attr, 2)
}

// Wr8 writes an 8-bit data value to memory.
func (m *Chunk) Wr8(adr uint, val uint8) Exception {
	m.mem[adr-m.start] = val
	return wrException(adr, m.attr, 1)
}

//-----------------------------------------------------------------------------

// Empty is an empty memory region.
type Empty struct {
	attr       Attribute // bitmask of attributes
	start, end uint      // address range
}

// NewEmpty allocates and returns an empty memory region.
func NewEmpty(start, size uint, attr Attribute) *Empty {
	return &Empty{
		attr:  attr,
		start: start,
		end:   start + size - 1,
	}
}

// In returns true if the adr, size is entirely within the empty region.
func (m *Empty) In(adr, size uint) bool {
	end := adr + size - 1
	return (adr >= m.start) && (end <= m.end)
}

// RdIns reads a 32-bit instruction from memory.
func (m *Empty) RdIns(adr uint) (uint, Exception) {
	return math.MaxUint32, rdInsException(adr, m.attr) | ExEmpty
}

// Rd64 reads a 64-bit data value from memory.
func (m *Empty) Rd64(adr uint) (uint64, Exception) {
	return math.MaxUint64, rdException(adr, m.attr, 8) | ExEmpty
}

// Rd32 reads a 32-bit data value from memory.
func (m *Empty) Rd32(adr uint) (uint32, Exception) {
	return math.MaxUint32, rdException(adr, m.attr, 4) | ExEmpty
}

// Rd16 reads a 16-bit data value from memory.
func (m *Empty) Rd16(adr uint) (uint16, Exception) {
	return math.MaxUint16, rdException(adr, m.attr, 2) | ExEmpty
}

// Rd8 reads an 8-bit data value from memory.
func (m *Empty) Rd8(adr uint) (uint8, Exception) {
	return math.MaxUint8, rdException(adr, m.attr, 1) | ExEmpty
}

// Wr64 writes a 64-bit data value to memory.
func (m *Empty) Wr64(adr uint, val uint64) Exception {
	return wrException(adr, m.attr, 8) | ExEmpty
}

// Wr32 writes a 32-bit data value to memory.
func (m *Empty) Wr32(adr uint, val uint32) Exception {
	return wrException(adr, m.attr, 4) | ExEmpty
}

// Wr16 writes a 16-bit data value to memory.
func (m *Empty) Wr16(adr uint, val uint16) Exception {
	return wrException(adr, m.attr, 2) | ExEmpty
}

// Wr8 writes an 8-bit data value to memory.
func (m *Empty) Wr8(adr uint, val uint8) Exception {
	return wrException(adr, m.attr, 1) | ExEmpty
}

//-----------------------------------------------------------------------------