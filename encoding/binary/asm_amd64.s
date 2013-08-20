// +build !appengine

TEXT    ·leUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    (AX),AX
    MOVW    AX,ret+24(FP)
    RET

TEXT    ·lePutUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    v+24(FP),BX
    MOVW    BX,(AX)
    RET

TEXT    ·leUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    MOVL    AX,ret+24(FP)
    RET

TEXT    ·lePutUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    v+24(FP),BX
    MOVL    BX, (AX)
    RET

TEXT    ·leUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    (AX),AX
    MOVQ    AX,ret+24(FP)
    RET

TEXT    ·lePutUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    v+24(FP),BX
    MOVQ    BX,(AX)
    RET

TEXT    ·beUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    XCHGB   AX,AH
    MOVW    AX,ret+24(FP)
    RET

TEXT    ·bePutUint16(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVW    v+24(FP),BX
    XCHGB   BX,BH
    MOVW    BX,(AX)
    RET

TEXT    ·beUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    (AX),AX
    BSWAPL  AX
    MOVL    AX,ret+24(FP)
    RET

TEXT    ·bePutUint32(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVL    v+24(FP),BX
    BSWAPL  BX
    MOVL    BX,(AX)
    RET

TEXT    ·beUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    (AX),AX
    BSWAPQ  AX
    MOVQ    AX,ret+24(FP)
    RET

TEXT    ·bePutUint64(SB),7,$0-32
    MOVQ    b+0(FP),AX
    MOVQ    v+24(FP),BX
    BSWAPQ  BX
    MOVQ    BX,(AX)
    RET
