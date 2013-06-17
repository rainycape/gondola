TEXT    ·methodPointer(SB),$72-56
// Copy string arg (16 bytes)
        MOVQ    op+0(FP),BX
        MOVQ    BX, (SP)
        MOVQ    op+8(FP), BX
        MOVQ    BX,8(SP)
// Copy reflect.Value (24 bytes)
        MOVQ    v+16(FP), BX
        MOVQ    BX, 16(SP)
        MOVQ    v+24(FP), BX
        MOVQ    BX, 24(SP)
        MOVQ    v+32(FP), BX
        MOVQ    BX, 32(SP)
// Copy index  (8 bytes)
        MOVQ    idx+40(FP), BX
        MOVQ    BX, 40(SP)
// Call reflect.methodReceiver
        CALL    reflect·methodReceiver(SB)
// Grab second return value
        MOVQ    56(SP), BX
        MOVQ    BX, fn+48(FP)
        RET
