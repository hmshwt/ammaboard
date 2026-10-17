// store_byte.go — store_byte rt, imm(rs): the low byte of rt lands at
// rs + imm, no alignment needed; posted like word stores.
package cpu

import (
	"ammaboard/internal/isa"
	"ammaboard/internal/machine"
)

func init() { register(isa.OpStoreByte, execStoreByte) }

func execStoreByte(c *CPU, in isa.Instruction, env machine.Env) {
	addr := c.Regs[in.Rs] + uint32(in.Imm)
	env.Logf("store_byte", "mem[0x%04x] <= r%d (byte %d)", addr, in.Rt, c.Regs[in.Rt]&0xff)
	env.WriteBusByte(addr, uint8(c.Regs[in.Rt]), machine.WriteSite{PC: c.at(), Word: c.fetchWord})
}
