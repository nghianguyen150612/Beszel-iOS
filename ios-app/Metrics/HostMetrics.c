#include "HostMetrics.h"
#include <mach/mach.h>
#include <stddef.h>

int32_t BeszelHostCPUTicks(uint32_t *ticks, uint32_t capacity, uint32_t *cores) {
    mach_port_t host = mach_host_self();
    natural_t processors = 0;
    processor_info_array_t info = NULL;
    mach_msg_type_number_t count = 0;
    kern_return_t result = host_processor_info(host, PROCESSOR_CPU_LOAD_INFO, &processors, &info, &count);
    mach_port_deallocate(mach_task_self(), host);
    if (result != KERN_SUCCESS) {
        return result;
    }

    if (info == NULL || processors == 0 || processors > capacity / 4 ||
        count != processors * CPU_STATE_MAX) {
        result = KERN_INVALID_VALUE;
    } else {
        for (natural_t i = 0; i < processors; ++i) {
            const integer_t *cpu = info + i * CPU_STATE_MAX;
            ticks[4 * i] = (uint32_t)cpu[CPU_STATE_USER];
            ticks[4 * i + 1] = (uint32_t)cpu[CPU_STATE_SYSTEM];
            ticks[4 * i + 2] = (uint32_t)cpu[CPU_STATE_NICE];
            ticks[4 * i + 3] = (uint32_t)cpu[CPU_STATE_IDLE];
        }
        *cores = processors;
    }

    if (info != NULL) {
        kern_return_t cleanup = vm_deallocate(mach_task_self(), (vm_address_t)info,
            (vm_size_t)count * sizeof(*info));
        if (result == KERN_SUCCESS) {
            result = cleanup;
        }
    }
    return result;
}

int32_t BeszelHostVMPages(uint64_t *freePages, uint64_t *speculativePages, uint64_t *inactivePages, uint64_t *pageSize) {
    mach_port_t host = mach_host_self();
    vm_size_t bytes = 0;
    kern_return_t result = host_page_size(host, &bytes);
    if (result == KERN_SUCCESS) {
        vm_statistics64_data_t stats = {0};
        mach_msg_type_number_t count = HOST_VM_INFO64_COUNT;
        result = host_statistics64(host, HOST_VM_INFO64, (host_info64_t)&stats, &count);
        if (result == KERN_SUCCESS) {
            if (count < HOST_VM_INFO64_REV0_COUNT) {
                result = KERN_INVALID_VALUE;
            } else {
                *freePages = stats.free_count;
                *speculativePages = stats.speculative_count;
                *inactivePages = stats.inactive_count;
                *pageSize = bytes;
            }
        }
    }
    mach_port_deallocate(mach_task_self(), host);
    return result;
}
