#include <runtime.h>

void ·goroutineId(int32 ret) {
    ret = g->goid;
    USED(&ret);
}
