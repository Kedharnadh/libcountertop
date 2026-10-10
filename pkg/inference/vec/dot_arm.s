//go:build arm && !noasm

#include "textflag.h"

// func dotVFP(a, b *float32, n int) float32
// Go's ARM assembler has no NEON mnemonics.
// VLDR has no post-indexed form; MOVF.P assembles without the writeback and rereads one element.
TEXT ·dotVFP(SB), NOSPLIT, $0-16
	MOVW	a+0(FP), R0
	MOVW	b+4(FP), R1
	MOVW	n+8(FP), R2

	MOVF	$(0.0), F0
	MOVF	$(0.0), F1
	MOVF	$(0.0), F2
	MOVF	$(0.0), F3

// An in-order core reads operands at issue: a register can be reloaded by the next instruction.
sixteen:
	CMP	$16, R2
	BLT	eight

	MOVF	0(R0), F4
	MOVF	0(R1), F5
	MOVF	4(R0), F6
	MOVF	4(R1), F7
	MOVF	8(R0), F8
	MOVF	8(R1), F9
	MOVF	12(R0), F10
	MOVF	12(R1), F11
	MOVF	16(R0), F12
	MOVF	16(R1), F13
	MOVF	20(R0), F14
	MOVF	20(R1), F15

	MULAF	F5, F4, F0
	MULAF	F7, F6, F1
	MULAF	F9, F8, F2
	MULAF	F11, F10, F3

	MOVF	24(R0), F4
	MOVF	24(R1), F5
	MOVF	28(R0), F6
	MOVF	28(R1), F7
	MOVF	32(R0), F8
	MOVF	32(R1), F9
	MOVF	36(R0), F10
	MOVF	36(R1), F11

	MULAF	F13, F12, F0
	MULAF	F15, F14, F1

	MOVF	40(R0), F12
	MOVF	40(R1), F13
	MOVF	44(R0), F14
	MOVF	44(R1), F15

	MULAF	F5, F4, F2
	MULAF	F7, F6, F3
	MULAF	F9, F8, F0
	MULAF	F11, F10, F1

	MOVF	48(R0), F4
	MOVF	48(R1), F5
	MOVF	52(R0), F6
	MOVF	52(R1), F7
	MOVF	56(R0), F8
	MOVF	56(R1), F9
	MOVF	60(R0), F10
	MOVF	60(R1), F11

	MULAF	F13, F12, F2
	MULAF	F15, F14, F3

	ADD	$64, R0
	ADD	$64, R1
	SUB	$16, R2

	MULAF	F5, F4, F0
	MULAF	F7, F6, F1
	MULAF	F9, F8, F2
	MULAF	F11, F10, F3
	B	sixteen

eight:
	CMP	$8, R2
	BLT	one

	MOVF	0(R0), F4
	MOVF	0(R1), F5
	MOVF	4(R0), F6
	MOVF	4(R1), F7
	MOVF	8(R0), F8
	MOVF	8(R1), F9
	MOVF	12(R0), F10
	MOVF	12(R1), F11
	MOVF	16(R0), F12
	MOVF	16(R1), F13
	MOVF	20(R0), F14
	MOVF	20(R1), F15

	MULAF	F5, F4, F0
	MULAF	F7, F6, F1
	MULAF	F9, F8, F2
	MULAF	F11, F10, F3

	MOVF	24(R0), F4
	MOVF	24(R1), F5
	MOVF	28(R0), F6
	MOVF	28(R1), F7

	MULAF	F13, F12, F0
	MULAF	F15, F14, F1

	ADD	$32, R0
	ADD	$32, R1
	SUB	$8, R2

	MULAF	F5, F4, F2
	MULAF	F7, F6, F3

one:
	CMP	$0, R2
	BEQ	fold

	MOVF	0(R0), F4
	MOVF	0(R1), F5
	MULAF	F5, F4, F0
	ADD	$4, R0
	ADD	$4, R1
	SUB	$1, R2
	B	one

fold:
	ADDF	F1, F0
	ADDF	F2, F0
	ADDF	F3, F0
	MOVF	F0, ret+12(FP)
	RET

// func dotNEON(a, b *float32, n int) float32
// NEON instructions are hand-encoded; VLD1 with no alignment hint accepts unaligned addresses.
TEXT ·dotNEON(SB), NOSPLIT, $0-16
	MOVW	a+0(FP), R0
	MOVW	b+4(FP), R1
	MOVW	n+8(FP), R2

	WORD	$0xF2800050 // VMOV.I32 q0, #0
	WORD	$0xF2802050 // VMOV.I32 q1, #0
	WORD	$0xF2804050 // VMOV.I32 q2, #0
	WORD	$0xF2806050 // VMOV.I32 q3, #0

wide:
	CMP	$16, R2
	BLT	half

	WORD	$0xF420828D // VLD1.32 {d8-d11}, [r0]!
	WORD	$0xF421C28D // VLD1.32 {d12-d15}, [r1]!
	WORD	$0xF460028D // VLD1.32 {d16-d19}, [r0]!
	WORD	$0xF461428D // VLD1.32 {d20-d23}, [r1]!
	WORD	$0xF2080D5C // VMLA.F32 q0, q4, q6
	WORD	$0xF20A2D5E // VMLA.F32 q1, q5, q7
	WORD	$0xF2004DF4 // VMLA.F32 q2, q8, q10
	WORD	$0xF2026DF6 // VMLA.F32 q3, q9, q11
	SUB	$16, R2
	B	wide

half:
	CMP	$8, R2
	BLT	reduce

	WORD	$0xF420828D // VLD1.32 {d8-d11}, [r0]!
	WORD	$0xF421C28D // VLD1.32 {d12-d15}, [r1]!
	WORD	$0xF2080D5C // VMLA.F32 q0, q4, q6
	WORD	$0xF20A2D5E // VMLA.F32 q1, q5, q7
	SUB	$8, R2

reduce:
	WORD	$0xF2000D42 // VADD.F32 q0, q0, q1
	WORD	$0xF2044D46 // VADD.F32 q2, q2, q3
	WORD	$0xF2000D44 // VADD.F32 q0, q0, q2
	WORD	$0xF2000D01 // VADD.F32 d0, d0, d1
	WORD	$0xF3000D00 // VPADD.F32 d0, d0, d0

rest:
	CMP	$0, R2
	BEQ	done

	MOVF	0(R0), F4
	MOVF	0(R1), F5
	MULAF	F5, F4, F0
	ADD	$4, R0
	ADD	$4, R1
	SUB	$1, R2
	B	rest

done:
	MOVF	F0, ret+12(FP)
	RET

// func dot4NEON(x, w0, w1, w2, w3 *float32, n int, out *[4]float32)
// n is a positive multiple of 4; the sums are stored to out, not added.
TEXT ·dot4NEON(SB), NOSPLIT, $0-28
	MOVW	x+0(FP), R0
	MOVW	w0+4(FP), R1
	MOVW	w1+8(FP), R2
	MOVW	w2+12(FP), R3
	MOVW	w3+16(FP), R4
	MOVW	n+20(FP), R5
	MOVW	out+24(FP), R7

	WORD	$0xF2800050 // VMOV.I32 q0, #0
	WORD	$0xF2802050 // VMOV.I32 q1, #0
	WORD	$0xF2804050 // VMOV.I32 q2, #0
	WORD	$0xF2806050 // VMOV.I32 q3, #0

four:
	WORD	$0xF4600A8D // VLD1.32 {d16-d17}, [r0]!
	WORD	$0xF4612A8D // VLD1.32 {d18-d19}, [r1]!
	WORD	$0xF4624A8D // VLD1.32 {d20-d21}, [r2]!
	WORD	$0xF4636A8D // VLD1.32 {d22-d23}, [r3]!
	WORD	$0xF4648A8D // VLD1.32 {d24-d25}, [r4]!
	WORD	$0xF2000DF2 // VMLA.F32 q0, q8, q9
	WORD	$0xF2002DF4 // VMLA.F32 q1, q8, q10
	WORD	$0xF2004DF6 // VMLA.F32 q2, q8, q11
	WORD	$0xF2006DF8 // VMLA.F32 q3, q8, q12
	SUB.S	$4, R5
	BNE	four

	WORD	$0xF3000D01 // VPADD.F32 d0, d0, d1
	WORD	$0xF3022D03 // VPADD.F32 d2, d2, d3
	WORD	$0xF3044D05 // VPADD.F32 d4, d4, d5
	WORD	$0xF3066D07 // VPADD.F32 d6, d6, d7
	WORD	$0xF3000D02 // VPADD.F32 d0, d0, d2
	WORD	$0xF3041D06 // VPADD.F32 d1, d4, d6
	WORD	$0xF4070A8F // VST1.32 {d0-d1}, [r7]
	RET
