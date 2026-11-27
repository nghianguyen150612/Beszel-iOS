#ifndef BESZEL_STOCK_HOST_METRICS_H
#define BESZEL_STOCK_HOST_METRICS_H

#include <stdint.h>

// CPU values are four cumulative 32-bit Mach ticks per logical processor,
// ordered user, system, nice, idle. The caller supplies the storage.
int32_t BeszelHostCPUTicks(uint32_t *ticks, uint32_t capacity, uint32_t *cores);
int32_t BeszelHostVMPages(uint64_t *freePages, uint64_t *speculativePages, uint64_t *inactivePages, uint64_t *pageSize);

#endif
