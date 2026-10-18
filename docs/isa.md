# ISA

## Rules
- Fixed 32-bit everything: words, registers, addresses, encodings. No 16→32-bit migration
  ever — MMU day needs real 32-bit VAs, cache day needs 64B lines.
- Load-store RISC, 8 registers `r0`–`r7`. `r0` is hardwired zero; `r7` is the fixed link
  register for `call`. No dedicated SP — the stack is RAM walked with ordinary registers.
- One status register: `IE` (bit 0), `PRIV` (bit 1, dormant until stage 3). No CSR file.
- Two's complement from day one.
- The ISA grows only when a chapter's pain demands it.
- Sugar (`nop mov ret li jump …`) gets real opcode slots but must always expand to
  existing primitives — never new semantics — and ships only after the learner has
  hand-expanded it once (day 5).
- Immediates are 16-bit two's complement, range `[-2^15, 2^15-1]`. `Encode` refuses wider
  values (they would not round-trip through `Decode`'s sign extension).
- Immediate reach rules for the whole roadmap: absolute labels and MMIO windows live in
  `[0, 2^15-1]` — the signed-16 ceiling is exactly the last byte of the 32KB RAM hard
  ceiling, so the whole board stays one-instruction addressable. Anything above `0x7fff`
  (future devices, big virtual addresses) is pointer-register territory — the same idiom
  MMU day teaches anyway.

## Encoding
```
[ opcode:7 | rd:3 | rs:3 | rt:3 | immediate:16 ]
```
- The verb fills the top 7 bits (31..25) and fields pack contiguously beneath it — a
  hexdump's leading digits still name the instruction. Opcodes 1..20 are defined;
  21..127 are reserved headroom (stage 3–5 verbs, sugar like `li`), and decode to
  nothing for now. Operand order is destination-first, max three
  operands, address syntax `offset(base)`.
- Reserved bits must be zero; the machine faults on nonzero reserved bits (it refuses
  nonsense). Zero-operand ops enforce this in both `Encode` and `Decode`.
- `isa.Specs` is the single source of truth (opcode, mnemonic, form). The assembler parses
  by form; the CPU registers executors by opcode; `internal/isa/parity_test.go` fails the
  build if the layers drift.

## Operand forms
| Form | Syntax | Instructions |
|---|---|---|
| zero | — | `halt` `allow_interrupts` `forbid_interrupts` `return_from_interrupt` |
| rrr | `rd, rs, rt` | `add` `subtract` `shift_left` `shift_right` `shift_right_arithmetic` `bit_and` `bit_or` |
| rri | `rd, rs, imm\|label` | `add_number` (label = absolute byte address; there is no `lui`) |
| load | `rd, imm(rs)` | `load_word` `load_byte` (zero-extended) |
| store | `rt, imm(rs)` | `store_word` `store_byte` (low byte) |
| branch | `rs, rt, target` | `jump_if_equal` `jump_if_not_equal` — byte offset from the next pc |
| call | `target` | `call` — return address → `r7` |
| goto | `rs` | `goto` — pc ← `r[rs]` |

Word accesses require 4-byte alignment (fault otherwise); byte accesses have no alignment
constraint. Shift amount is `rt & 31`. There is no `cmp`/flags — branches compare
registers directly.

## Growth stages
| Stage | Days | Adds | Status |
|---|---|---|---|
| 1 "boot ten" | 4–5 | `load_word store_word add add_number subtract jump_if_equal jump_if_not_equal call goto halt` (opcodes 1–10) | done |
| 2 | 7–8 | `load_byte store_byte shift_left shift_right shift_right_arithmetic bit_and bit_or allow_interrupts forbid_interrupts return_from_interrupt` (opcodes 11–20) | done |
| 3 | 10–11 | `flush_tlb`; `PRIV` bit; trap vector + fault registers | planned |
| 4 | 13–14 | `multiply` | planned |
| 5 | 17 | `swap_atomic add_atomic` | planned |
| 6 | 22 | nothing — P/E scheduling is pure software (that IS the lesson) | — |

21 encoded instructions by day 11. The memory system (base+limit, MMU, cache) adds almost
no instructions — true of real hardware, and an explicit teaching point. Devices are just
addresses.
