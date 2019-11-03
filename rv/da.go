//-----------------------------------------------------------------------------
/*

RISC-V Disassembler

*/
//-----------------------------------------------------------------------------

package rv

import (
	"fmt"
	"strings"
	"unsafe"
)

//-----------------------------------------------------------------------------

func daNone(mneumonic string, adr, ins uint32) (string, string) {
	return mneumonic, "TODO"
}

//-----------------------------------------------------------------------------

// SymbolTable maps an address to a symbol.
type SymbolTable map[uint32]string

// Disassembly returns the result of the disassembler call.
type Disassembly struct {
	Dump        string // address and memory bytes
	Symbol      string // symbol for the address (if any)
	Instruction string // instruction decode
	Comment     string // useful comment
	N           int    // length in bytes of decode
}

func (da *Disassembly) String() string {
	s := make([]string, 2)
	s[0] = fmt.Sprintf("%-16s %8s %-13s", da.Dump, da.Symbol, da.Instruction)
	if da.Comment != "" {
		s[1] = fmt.Sprintf(" ; %s", da.Comment)
	}
	return strings.Join(s, "")
}

//-----------------------------------------------------------------------------

func daDump(adr, ins uint32) string {
	return fmt.Sprintf("%08x: %08x", adr, ins)
}

func daSymbol(adr uint32, st SymbolTable) string {
	if st != nil {
		return st[adr]
	}
	return ""
}

func daInstruction(adr, ins uint32) (string, string) {
	return "", ""
}

//-----------------------------------------------------------------------------

// Disassemble a RISC-V instruction at the address.
func (m *RV) Disassemble(adr uint32, st SymbolTable) *Disassembly {

	ins := m.Mem.Read32(adr)

	var instruction, comment string

	for _, ii := range m.isa.instruction {
		if ins&ii.mask == ii.val {
			instruction, comment = ii.decode.da(ii.mneumonic, adr, ins)
			break
		}
	}

	return &Disassembly{
		Dump:        daDump(adr, ins),
		Symbol:      daSymbol(adr, st),
		Instruction: instruction,
		Comment:     comment,
		N:           int(unsafe.Sizeof(ins)),
	}
}

//-----------------------------------------------------------------------------
