package agent

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/shirou/gopsutil/v4/disk"
)

var errIOSDiskIO = errors.New("physical disk I/O unavailable on iOS")

func unavailableIOSDiskIO(...string) (map[string]disk.IOCountersStat, error) {
	return nil, errIOSDiskIO
}

// iOS usage tracks actual mountpoints, independently of physical I/O identities.
// Default to /; /private/var is not assumed to be an equivalent capacity source.
// Related mounts are not automatically added. Extras require configuration.
func (d *diskDiscovery) addIOSFilesystems(extra string) {
	root := disk.PartitionStat{Mountpoint: "/"}
	for _, p := range d.partitions {
		if p.Mountpoint == "/" {
			root = p
			break
		}
	}
	if d.ctx.filesystem != "" {
		for _, p := range d.partitions {
			if p.Mountpoint == d.ctx.filesystem || p.Device == d.ctx.filesystem || filepath.Base(p.Device) == d.ctx.filesystem {
				root = p
				break
			}
		}
	}
	add := func(p disk.PartitionStat, isRoot bool, name string) {
		key := p.Device
		if key == "" {
			key = p.Mountpoint
		}
		for _, existing := range d.agent.fsStats {
			if existing.Mountpoint == p.Mountpoint {
				return
			}
		}
		if _, exists := d.agent.fsStats[key]; exists {
			return
		}
		d.agent.fsStats[key] = &system.FsStats{Root: isRoot, Mountpoint: p.Mountpoint, Name: name}
	}
	add(root, true, d.ctx.filesystemName)
	for _, entry := range strings.Split(extra, ",") {
		requested, name := parseFilesystemEntry(entry)
		if requested == "" {
			continue
		}
		found := false
		for _, p := range d.partitions {
			if requested == p.Mountpoint || requested == p.Device || requested == filepath.Base(p.Device) {
				add(p, false, name)
				found = true
				break
			}
		}
		if !found {
			if usage, err := d.usageFn(requested); err == nil && usage != nil && usage.Total > 0 {
				add(disk.PartitionStat{Mountpoint: requested}, false, name)
			}
		}
	}
}
