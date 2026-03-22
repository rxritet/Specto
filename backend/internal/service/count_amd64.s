#include "textflag.h"

// func countStatuses(statuses []byte) (todo, inProgress, done int)
//
// SSE2 + POPCNT implementation.
// Processes 16 bytes per iteration via PCMPEQB / PMOVMSKB / POPCNTL.
// Falls through to a scalar tail for the remaining 0-15 bytes.
//
// Requires POPCNT (available on all amd64 CPUs since ~2008, GOAMD64>=v2).
TEXT ·countStatuses(SB), NOSPLIT, $0-48
	MOVQ	statuses+0(FP), SI          // base pointer
	MOVQ	statuses_len+8(FP), CX      // length

	// Zero accumulators.
	XORQ	R8, R8                       // todo
	XORQ	R9, R9                       // inProgress
	XORQ	R10, R10                     // done

	// ---- build comparison vectors ----
	// X3 = {0x00 × 16}  (todo)
	PXOR	X3, X3

	// X4 = {0x01 × 16}  (in_progress)
	PXOR	X4, X4
	PCMPEQB	X5, X5                       // X5 = {0xFF × 16}
	PSUBB	X5, X4                       // 0x00 − 0xFF wraps to 0x01

	// X5 = {0x02 × 16}  (done)
	MOVOU	X4, X5
	PADDB	X4, X5                       // 0x01 + 0x01 = 0x02

	// ---- SIMD loop: 16 bytes per iteration ----
	CMPQ	CX, $16
	JL	tail

simd:
	MOVOU	(SI), X0                     // load 16 status bytes

	// count todo (== 0)
	MOVOU	X0, X1
	PCMPEQB	X3, X1                       // 0xFF where byte == 0
	PMOVMSKB X1, AX
	POPCNTL	AX, AX
	ADDQ	AX, R8

	// count in_progress (== 1)
	MOVOU	X0, X1
	PCMPEQB	X4, X1
	PMOVMSKB X1, AX
	POPCNTL	AX, AX
	ADDQ	AX, R9

	// count done (== 2)
	MOVOU	X0, X1
	PCMPEQB	X5, X1
	PMOVMSKB X1, AX
	POPCNTL	AX, AX
	ADDQ	AX, R10

	ADDQ	$16, SI
	SUBQ	$16, CX
	CMPQ	CX, $16
	JGE	simd

	// ---- scalar tail: 0-15 remaining bytes ----
tail:
	TESTQ	CX, CX
	JZ	end

scalar:
	MOVBQZX	(SI), DX

	TESTQ	DX, DX
	JNE	notTodo
	INCQ	R8
	JMP	advance

notTodo:
	CMPQ	DX, $1
	JNE	notInProg
	INCQ	R9
	JMP	advance

notInProg:
	CMPQ	DX, $2
	JNE	advance
	INCQ	R10

advance:
	INCQ	SI
	DECQ	CX
	JNZ	scalar

end:
	MOVQ	R8, todo+24(FP)
	MOVQ	R9, inProgress+32(FP)
	MOVQ	R10, done+40(FP)
	RET
