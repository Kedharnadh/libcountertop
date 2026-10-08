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
