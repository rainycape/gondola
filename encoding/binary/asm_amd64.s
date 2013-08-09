// +build !appengine

TEXT    ·littleEndian·Uint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    (AX),AX
    MOVW    AX,ret+24(FP)
    RET

TEXT    ·littleEndian·PutUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    v+24(FP),BX
    MOVW    BX,(AX)
    RET

TEXT    ·littleEndian·Uint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    MOVL    AX,ret+24(FP)
    RET

TEXT    ·littleEndian·PutUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    v+24(FP),BX
    MOVL    BX, (AX)
    RET

TEXT    ·littleEndian·Uint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    (AX),AX
    MOVQ    AX,ret+24(FP)
    RET

TEXT    ·littleEndian·PutUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    v+24(FP),BX
    MOVQ    BX,(AX)
    RET

TEXT    ·bigEndian·Uint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    XCHGB   AX,AH
    MOVW    AX,ret+24(FP)
    RET

TEXT    ·bigEndian·PutUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    v+24(FP),BX
    XCHGB   BX,BH
    MOVW    BX,(AX)
    RET

TEXT    ·bigEndian·Uint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    BSWAPL  AX
    MOVL    AX,ret+24(FP)
    RET

TEXT    ·bigEndian·PutUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    v+24(FP),BX
    BSWAPL  BX
    MOVL    BX,(AX)
    RET

TEXT    ·bigEndian·Uint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    (AX),AX
    BSWAPQ  AX
    MOVQ    AX,ret+24(FP)
    RET

TEXT    ·bigEndian·PutUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    v+24(FP),BX
    BSWAPQ  BX
    MOVQ    BX,(AX)
    RET
