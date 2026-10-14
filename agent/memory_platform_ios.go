//go:build ios

package agent

/*
#include <mach/mach.h>
static kern_return_t beszel_vm_statistics(vm_statistics_data_t *stats) {
 mach_port_t host = mach_host_self();
 mach_msg_type_number_t count = HOST_VM_INFO_COUNT;
 kern_return_t result = host_statistics(host, HOST_VM_INFO, (host_info_t)stats, &count);
 mach_port_deallocate(mach_task_self(), host);
 return result;
}
*/
import "C"

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/mem"
	"golang.org/x/sys/unix"
)

func init() { memoryVirtualMemory = readIOSMemory }

func readIOSMemory() (*mem.VirtualMemoryStat, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return nil, err
	}
	var pages C.vm_statistics_data_t
	if status := C.beszel_vm_statistics(&pages); status != C.KERN_SUCCESS {
		return nil, fmt.Errorf("iOS host_statistics error=%d", status)
	}
	return iosMemoryFromPages(total, uint64(unix.Getpagesize()), uint64(pages.free_count),
		uint64(pages.active_count), uint64(pages.inactive_count), uint64(pages.wire_count))
}
