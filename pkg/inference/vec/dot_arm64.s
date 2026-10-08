//go:build arm64 && !noasm

#include "textflag.h"

// func dotNEON(a, b *float32, n int) float32
// The Go arm64 assembler's only vector float arithmetic is FMLA.
TEXT ·dotNEON(SB), NOSPLIT, $0-28
	MOVD	a+0(FP), R0
	MOVD	b+8(FP), R1
	MOVD	n+16(FP), R2

	VEOR	V0.B16, V0.B16, V0.B16
	VEOR	V1.B16, V1.B16, V1.B16
	VEOR	V2.B16, V2.B16, V2.B16
	VEOR	V3.B16, V3.B16, V3.B16

	CMP	$16, R2
	BLT	four

	VLD1.P	64(R0), [V4.S4, V5.S4, V6.S4, V7.S4]
	VLD1.P	64(R1), [V8.S4, V9.S4, V10.S4, V11.S4]
	SUB	$16, R2

	// R2 is what has not been loaded.
pipe:
	CMP	$16, R2
	BLT	drainfirst

	VLD1.P	64(R0), [V20.S4, V21.S4, V22.S4, V23.S4]
	VLD1.P	64(R1), [V24.S4, V25.S4, V26.S4, V27.S4]
	SUB	$16, R2

	VFMLA	V8.S4, V4.S4, V0.S4
	VFMLA	V9.S4, V5.S4, V1.S4
	VFMLA	V10.S4, V6.S4, V2.S4
	VFMLA	V11.S4, V7.S4, V3.S4

	CMP	$16, R2
	BLT	drainsecond

	VLD1.P	64(R0), [V4.S4, V5.S4, V6.S4, V7.S4]
	VLD1.P	64(R1), [V8.S4, V9.S4, V10.S4, V11.S4]
	SUB	$16, R2

	VFMLA	V24.S4, V20.S4, V0.S4
	VFMLA	V25.S4, V21.S4, V1.S4
	VFMLA	V26.S4, V22.S4, V2.S4
	VFMLA	V27.S4, V23.S4, V3.S4

	B	pipe

drainsecond:
	VFMLA	V24.S4, V20.S4, V0.S4
	VFMLA	V25.S4, V21.S4, V1.S4
	VFMLA	V26.S4, V22.S4, V2.S4
	VFMLA	V27.S4, V23.S4, V3.S4
	B	fold

drainfirst:
	VFMLA	V8.S4, V4.S4, V0.S4
	VFMLA	V9.S4, V5.S4, V1.S4
	VFMLA	V10.S4, V6.S4, V2.S4
	VFMLA	V11.S4, V7.S4, V3.S4

fold:
	FMOVS	$(1.0), F16
	VDUP	V16.S[0], V16.S4
	VFMLA	V16.S4, V1.S4, V0.S4
	VFMLA	V16.S4, V2.S4, V0.S4
	VFMLA	V16.S4, V3.S4, V0.S4

four:
	CMP	$4, R2
	BLT	across
	VLD1.P	16(R0), [V4.S4]
	VLD1.P	16(R1), [V8.S4]
	VFMLA	V8.S4, V4.S4, V0.S4
	SUB	$4, R2
	B	four

	// Writing F0 writes V0's low lane and clears the rest of it.
across:
	VDUP	V0.S[1], V17.S4
	VDUP	V0.S[2], V18.S4
	VDUP	V0.S[3], V19.S4
	FADDS	F17, F0, F0
	FADDS	F18, F0, F0
	FADDS	F19, F0, F0

	CBZ	R2, done

one:
	FMOVS	(R0), F1
	FMOVS	(R1), F2
	FMADDS	F2, F0, F1, F0
	ADD	$4, R0
	ADD	$4, R1
	SUB	$1, R2
	CBNZ	R2, one

done:
	FMOVS	F0, ret+24(FP)
	RET
