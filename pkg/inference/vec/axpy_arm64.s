//go:build arm64 && !noasm

#include "textflag.h"

// func axpyNEON(dst, x *float32, n int, gain float32)
// g is the goroutine register and will not parse as an argument name.
// R0 both reads and writes dst; the store advances it.
TEXT ·axpyNEON(SB), NOSPLIT, $0-28
	MOVD	dst+0(FP), R0
	MOVD	x+8(FP), R1
	MOVD	n+16(FP), R2
	FMOVS	gain+24(FP), F16
	VDUP	V16.S[0], V16.S4

sixteen:
	CMP	$16, R2
	BLT	four

	VLD1.P	64(R1), [V4.S4, V5.S4, V6.S4, V7.S4]
	VLD1	(R0), [V8.S4, V9.S4, V10.S4, V11.S4]

	VFMLA	V16.S4, V4.S4, V8.S4
	VFMLA	V16.S4, V5.S4, V9.S4
	VFMLA	V16.S4, V6.S4, V10.S4
	VFMLA	V16.S4, V7.S4, V11.S4

	VST1.P	[V8.S4, V9.S4, V10.S4, V11.S4], 64(R0)
	SUB	$16, R2
	B	sixteen

four:
	CMP	$4, R2
	BLT	one

	VLD1.P	16(R1), [V4.S4]
	VLD1	(R0), [V8.S4]
	VFMLA	V16.S4, V4.S4, V8.S4
	VST1.P	[V8.S4], 16(R0)
	SUB	$4, R2
	B	four

one:
	CBZ	R2, done

single:
	FMOVS	(R0), F0
	FMOVS	(R1), F1
	FMADDS	F16, F0, F1, F0
	FMOVS	F0, (R0)
	ADD	$4, R0
	ADD	$4, R1
	SUB	$1, R2
	CBNZ	R2, single

done:
	RET
