//go:build testing

package agent

import (
	"errors"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestP008IOSCPUModel(t *testing.T) {
	for _, machine := range []string{"iPad4,1", "iPad4,2", "iPad4,3", "iPad4,4", "iPad4,5", "iPad4,6", "iPad4,7", "iPad4,8", "iPad4,9", "iPhone6,1", "iPhone6,2"} {
		t.Run(machine, func(t *testing.T) { assert.Equal(t, "Apple A7", iosCPUModel(machine)) })
	}
	for _, machine := range []string{"iPad4,10", "iPad4,999", "iPad5,1", "iPhone16,1"} {
		assert.Equal(t, "Apple SoC ("+machine+")", iosCPUModel(machine))
	}
	assert.Empty(t, iosCPUModel(""))
}
func TestP008IOSMetadataSources(t *testing.T) {
	valid := []byte(`<?xml version="1.0"?><plist version="1.0"><dict><key>ProductVersion</key><string>12.5.7</string></dict></plist>`)
	for _, tt := range []struct {
		name, missing, arch string
		cores, threads      uint32
		file                []byte
		fileErr             bool
		wantOS              string
	}{
		{"reference", "", "arm64", 2, 2, valid, false, "iOS 12.5.7"},
		{"missing hostname", "kern.hostname", "arm64", 2, 2, valid, false, "iOS 12.5.7"},
		{"missing kernel", "kern.osrelease", "arm64", 2, 2, valid, false, "iOS 12.5.7"},
		{"missing machine", "hw.machine", "arm64", 2, 2, valid, false, "iOS 12.5.7"},
		{"missing counts", "counts", "arm64", 0, 0, valid, false, "iOS 12.5.7"},
		{"invalid counts", "", "arm64", ^uint32(0), ^uint32(0), valid, false, "iOS 12.5.7"},
		{"threads smaller", "", "arm64", 4, 2, valid, false, "iOS 12.5.7"},
		{"malformed plist", "", "arm64", 2, 2, []byte("not a plist"), false, "iOS"},
		{"missing version", "", "arm64", 2, 2, []byte(`<plist><dict/></plist>`), false, "iOS"},
		{"missing file", "", "arm64", 2, 2, nil, true, "iOS"},
		{"non arm64 fixture", "", "amd64", 2, 2, valid, false, "iOS 12.5.7"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := system.Details{OsName: "macOS", CpuModel: "old", Hostname: "old", Kernel: "old", Cores: 8, Threads: 8}
			source := iosMetadataSource{arch: tt.arch,
				sysctl: func(key string) (string, error) {
					if key == tt.missing {
						return "", errors.New("missing")
					}
					return map[string]string{"hw.machine": "iPad4,4", "kern.hostname": "fixture-ipad", "kern.osrelease": "18.7.0"}[key], nil
				},
				count: func(key string) (uint32, error) {
					if tt.missing == "counts" {
						return 0, errors.New("missing")
					}
					if key == "hw.physicalcpu" {
						return tt.cores, nil
					}
					return tt.threads, nil
				},
				readFile: func(path string) ([]byte, error) {
					assert.Equal(t, "/System/Library/CoreServices/SystemVersion.plist", path)
					if tt.fileErr {
						return nil, errors.New("missing")
					}
					return tt.file, nil
				},
			}
			applyIOSSystemDetails(&d, source)
			assert.Equal(t, system.Darwin, d.Os)
			assert.Equal(t, tt.wantOS, d.OsName)
			assert.Equal(t, tt.arch, d.Arch)
			if tt.missing == "hw.machine" {
				assert.Empty(t, d.CpuModel)
			} else {
				assert.Equal(t, "Apple A7", d.CpuModel)
			}
			if tt.missing == "kern.hostname" {
				assert.Empty(t, d.Hostname)
			} else {
				assert.Equal(t, "fixture-ipad", d.Hostname)
			}
			if tt.missing == "kern.osrelease" {
				assert.Empty(t, d.Kernel)
			} else {
				assert.Equal(t, "18.7.0", d.Kernel)
			}
			assert.LessOrEqual(t, d.Cores, d.Threads)
			if tt.name == "invalid counts" || tt.name == "missing counts" {
				assert.Zero(t, d.Cores)
				assert.Zero(t, d.Threads)
			}
		})
	}
}
