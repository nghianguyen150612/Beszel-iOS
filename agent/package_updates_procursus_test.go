//go:build testing

package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProcursusProbe struct {
	configOutput  string
	statusOutput  string
	lookPathValue string
	lookPathErr   error
	openErr       error
	commands      []string
}

func (p *fakeProcursusProbe) stat(path string) (os.FileInfo, error)      { return os.Stat(path) }
func (p *fakeProcursusProbe) readDir(path string) ([]os.DirEntry, error) { return os.ReadDir(path) }
func (p *fakeProcursusProbe) open(path string) (procursusReadCloser, error) {
	if p.openErr != nil {
		return nil, p.openErr
	}
	return os.Open(path)
}
func (p *fakeProcursusProbe) evalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}
func (p *fakeProcursusProbe) lookPath(string) (string, error) { return p.lookPathValue, p.lookPathErr }
func (p *fakeProcursusProbe) run(_ context.Context, executable string, args ...string) (string, error) {
	p.commands = append(p.commands, filepath.Base(executable)+" "+strings.Join(args, " "))
	switch filepath.Base(executable) {
	case "apt-get":
		if len(args) == 1 && args[0] == "--version" {
			return "apt 2.9.8 (iphoneos-arm64)\n", nil
		}
	case "apt-config":
		if len(args) == 1 && args[0] == "dump" {
			return p.configOutput, nil
		}
	case "dpkg-query":
		if len(args) == 4 && args[0] == "-W" && args[1] == "-f=${Package} ${Status}\\n" && args[2] == "apt" && args[3] == "procursus-keyring" {
			return p.statusOutput, nil
		}
	}
	return "", errors.New("unexpected probe argv")
}

func makeProcursusFixture(t *testing.T, root, binRel string) (string, string) {
	t.Helper()
	binDir := filepath.Join(root, filepath.FromSlash(binRel))
	aptRoot := root
	if strings.HasSuffix(filepath.ToSlash(binRel), "var/jb/usr/bin") {
		aptRoot = filepath.Join(root, "var/jb")
	}
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	for _, name := range []string{"apt-get", "apt-config", "dpkg-query"} {
		require.NoError(t, os.WriteFile(filepath.Join(binDir, name), []byte("fixture executable"), 0o755))
	}
	for _, path := range []string{
		filepath.Join(aptRoot, "etc/apt/sources.list.d"),
		filepath.Join(aptRoot, "etc/apt/apt.conf.d"),
		filepath.Join(aptRoot, "var/lib/apt/lists"),
		filepath.Join(aptRoot, "var/lib/dpkg"),
	} {
		require.NoError(t, os.MkdirAll(path, 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(aptRoot, "var/lib/dpkg/status"), []byte("Package: fixture\nStatus: install ok installed\nVersion: 1\n\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(aptRoot, "var/lib/apt/lists/example_dists_stable_main_binary-iphoneos-arm_Packages"), []byte("Package: fixture\nVersion: 2\n\n"), 0o644))
	return binDir, aptRoot
}

func makeProcursusProbe(aptRoot string) *fakeProcursusProbe {
	return &fakeProcursusProbe{
		configOutput: "Dir \"" + aptRoot + "\";\n" +
			"Dir::State \"var/lib/apt\";\n" +
			"Dir::State::status \"" + filepath.Join(aptRoot, "var/lib/dpkg/status") + "\";\n" +
			"Dir::State::lists \"" + filepath.Join(aptRoot, "var/lib/apt/lists") + "\";\n" +
			"Dir::Etc \"" + filepath.Join(aptRoot, "etc/apt") + "\";\n" +
			"Dir::Etc::sourcelist \"" + filepath.Join(aptRoot, "etc/apt/sources.list") + "\";\n" +
			"Dir::Etc::sourceparts \"" + filepath.Join(aptRoot, "etc/apt/sources.list.d") + "\";\n" +
			"Dir::Etc::main \"" + filepath.Join(aptRoot, "etc/apt/apt.conf") + "\";\n" +
			"Dir::Etc::parts \"" + filepath.Join(aptRoot, "etc/apt/apt.conf.d") + "\";\n",
		statusOutput: "apt install ok installed\nprocursus-keyring install ok installed\n",
		lookPathErr:  os.ErrNotExist,
	}
}

func TestDetectProcursusAPTLayouts(t *testing.T) {
	for _, tt := range []struct {
		name, binRel, wantKind string
	}{
		{"rootful", "usr/bin", "rootful"},
		{"rootless", "var/jb/usr/bin", "rootless"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			_, aptRoot := makeProcursusFixture(t, root, tt.binRel)
			probe := makeProcursusProbe(aptRoot)
			binaryDirs := []string{filepath.Join(root, "usr/bin"), filepath.Join(root, "var/jb/usr/bin")}

			got := detectProcursusAPT(context.Background(), probe, binaryDirs)

			require.Equal(t, procursusSupported, got.status)
			assert.Equal(t, tt.wantKind, got.layout.kind)
			assert.Equal(t, filepath.Join(aptRoot, "var/lib/dpkg/status"), got.layout.statusPath)
			assert.Equal(t, filepath.Join(aptRoot, "var/lib/apt/lists"), got.layout.listsPath)
			assert.Equal(t, filepath.Join(aptRoot, "etc/apt/sources.list.d"), got.layout.sourceParts)
			assert.Equal(t, []string{
				"apt-get --version",
				"apt-config dump",
				"dpkg-query -W -f=${Package} ${Status}\\n apt procursus-keyring",
			}, probe.commands)
		})
	}
}

func TestDetectProcursusAPTUnavailableCases(t *testing.T) {
	t.Run("no package manager", func(t *testing.T) {
		root := t.TempDir()
		probe := makeProcursusProbe(root)
		probe.statusOutput = ""
		got := detectProcursusAPT(context.Background(), probe, []string{filepath.Join(root, "usr/bin"), filepath.Join(root, "var/jb/usr/bin")})
		assert.Equal(t, procursusNoManager, got.status)
	})

	t.Run("incomplete executables", func(t *testing.T) {
		root := t.TempDir()
		binDir := filepath.Join(root, "usr/bin")
		require.NoError(t, os.MkdirAll(binDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(binDir, "apt-get"), []byte("partial"), 0o755))
		probe := makeProcursusProbe(root)
		got := detectProcursusAPT(context.Background(), probe, []string{binDir})
		assert.Equal(t, procursusIncomplete, got.status)
	})

	t.Run("unsupported apt variant without Procursus keyring", func(t *testing.T) {
		root := t.TempDir()
		binDir, aptRoot := makeProcursusFixture(t, root, "usr/bin")
		probe := makeProcursusProbe(aptRoot)
		probe.statusOutput = "apt install ok installed\n"
		got := detectProcursusAPT(context.Background(), probe, []string{binDir})
		assert.Equal(t, procursusUnsupported, got.status)
	})

	t.Run("missing cached package indexes", func(t *testing.T) {
		root := t.TempDir()
		binDir, aptRoot := makeProcursusFixture(t, root, "usr/bin")
		require.NoError(t, os.Remove(filepath.Join(aptRoot, "var/lib/apt/lists/example_dists_stable_main_binary-iphoneos-arm_Packages")))
		probe := makeProcursusProbe(aptRoot)
		got := detectProcursusAPT(context.Background(), probe, []string{binDir})
		assert.Equal(t, procursusIncomplete, got.status)
	})

	t.Run("conflicting installations", func(t *testing.T) {
		root := t.TempDir()
		rootful, _ := makeProcursusFixture(t, filepath.Join(root, "rootful"), "usr/bin")
		rootless, _ := makeProcursusFixture(t, filepath.Join(root, "rootless"), "usr/bin")
		probe := makeProcursusProbe(filepath.Join(root, "rootful"))
		got := detectProcursusAPT(context.Background(), probe, []string{rootful, rootless})
		assert.Equal(t, procursusAmbiguous, got.status)
	})

	t.Run("apt-get outside supported layouts", func(t *testing.T) {
		root := t.TempDir()
		probe := makeProcursusProbe(root)
		probe.lookPathValue = filepath.Join(root, "opt/procursus/usr/bin/apt-get")
		probe.lookPathErr = nil
		got := detectProcursusAPT(context.Background(), probe, []string{filepath.Join(root, "usr/bin")})
		assert.Equal(t, procursusUnsupported, got.status)
	})

	t.Run("insufficient status-file permission", func(t *testing.T) {
		root := t.TempDir()
		binDir, aptRoot := makeProcursusFixture(t, root, "usr/bin")
		probe := makeProcursusProbe(aptRoot)
		probe.openErr = os.ErrPermission
		got := detectProcursusAPT(context.Background(), probe, []string{binDir})
		assert.Equal(t, procursusInsufficientPermission, got.status)
	})
}

func TestParseProcursusAPTOutput(t *testing.T) {
	// This is a sanitized, synthetic fixture shaped from apt-get's documented
	// simulation grammar; it is not captured from a physical Procursus device.
	output := readPackageUpdatesTestData(t, "apt_procursus_simulate.txt")
	packages, err := parseProcursusAPTOutput(output)
	require.NoError(t, err)
	assert.Equal(t, []system.PackageUpdate{
		{Name: "bar++-2", Current: "1.0-1", Available: "1.0-2"},
		{Name: "foo:iphoneos-arm", Current: "0:1.0-1", Available: "0:1.0-2"},
		{Name: "libfoo:arm64", Current: "1:2.0~rc1-3", Available: "2:2.0+dfsg-1"},
		{Name: "very-long-procursus-package-name-2026", Current: "1.2.3~beta-1", Available: "1.2.3+final-1"},
	}, packages)
}

func TestProcursusAPTParserRequiresCompleteOutput(t *testing.T) {
	tests := []struct {
		name, output string
	}{
		{"localized output", "0 mis à jour, 0 nouvellement installés\n"},
		{"partial output after summary", "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\nInst truncated [1.0\n"},
		{"invalid package name", "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\nInst ../escape [1.0] (2.0 repo)\n"},
		{"missing candidate", "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\nInst package-name [1.0]\n"},
		{"summary without parsed updates", "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n"},
		{"conflicting duplicate", "2 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\nInst duplicate [1.0] (2.0 repo)\nInst duplicate [1.0] (3.0 repo)\n"},
		{"duplicate summary", "0 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n0 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProcursusAPTOutput(tt.output)
			assert.Error(t, err)
		})
	}

	_, err := parseProcursusAPTOutput(strings.Repeat("x", procursusQueryOutputLimit+1))
	assert.ErrorIs(t, err, errProcursusProbeOutputTooLarge)
}

func TestProcursusAPTParserZeroOneDuplicateAndDowngradeSemantics(t *testing.T) {
	zero, err := parseProcursusAPTOutput("Reading package lists...\n0 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n")
	require.NoError(t, err)
	assert.NotNil(t, zero)
	assert.Empty(t, zero)

	output := "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n" +
		"Inst one-package [1.0] (1.1 repo)\n" +
		"Inst one-package [1.0] (1.1 repo)\n" +
		"Inst downgrade-package [2:1.0-1] (1:99.0-1 repo)\n" +
		"Inst reinstall-package [1.0] (1.0 repo)\n" +
		"Remv obsolete-package [1.0]\n"
	packages, err := parseProcursusAPTOutput(output)
	require.NoError(t, err)
	assert.Equal(t, []system.PackageUpdate{{Name: "one-package", Current: "1.0", Available: "1.1"}}, packages)
}

func TestCompareDpkgVersions(t *testing.T) {
	tests := []struct {
		left, right string
		want        int
	}{
		{"2:1.0-1", "1:99.0-1", 1},
		{"1.0~beta-1", "1.0-1", -1},
		{"1.0-1", "1.0-2", -1},
		{"1.0+dfsg-1", "1.0~rc1-1", 1},
		{"0:1.0-1", "1.0-1", 0},
		{"1.0000000000000000000000002", "1.0000000000000000000000001", 1},
	}
	for _, tt := range tests {
		t.Run(tt.left+" vs "+tt.right, func(t *testing.T) {
			got, err := compareDpkgVersions(tt.left, tt.right)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	_, err := compareDpkgVersions("not:a:version", "1.0")
	assert.Error(t, err)
}

func TestCheckProcursusAPTReadOnlyArgvAndResults(t *testing.T) {
	layout := procursusAPTLayout{aptGet: "/var/jb/usr/bin/apt-get"}
	output := readPackageUpdatesTestData(t, "apt_procursus_simulate.txt")
	var gotExecutable string
	var gotArgs []string
	result, err := checkProcursusAPTWith(context.Background(), layout, func(_ context.Context, executable string, args ...string) (string, error) {
		gotExecutable = executable
		gotArgs = append([]string(nil), args...)
		return output, nil
	})
	require.NoError(t, err)
	assert.Equal(t, layout.aptGet, gotExecutable)
	assert.Equal(t, []string{
		"-s", "--no-download", "-o", "Debug::NoLocking=1", "-o", "Dir::Cache::pkgcache=", "-o", "Dir::Cache::srcpkgcache=", "dist-upgrade",
	}, gotArgs)
	assert.Equal(t, []uint16{4}, result.counts)
	assert.Len(t, result.packages, 4)
	assert.False(t, result.securityKnown)
	for _, pkg := range result.packages {
		assert.False(t, pkg.Security)
	}

	zero, err := checkProcursusAPTWith(context.Background(), layout, func(context.Context, string, ...string) (string, error) {
		return "0 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n", nil
	})
	require.NoError(t, err)
	assert.Equal(t, []uint16{0}, zero.counts, "verified zero remains distinct from unknown nil counts")
	assert.NotNil(t, zero.packages)
	assert.False(t, zero.securityKnown)
}

func TestProcursusAPTFailuresRemainUnknown(t *testing.T) {
	layout := procursusAPTLayout{aptGet: "/var/jb/usr/bin/apt-get"}
	for _, tt := range []struct {
		name   string
		output string
		err    error
	}{
		{"unsupported APT flag", "", errors.New("unsupported option --no-download")},
		{"missing package indexes", "", errors.New("package lists unavailable")},
		{"corrupt package database", "", errors.New("dpkg status database is corrupt")},
		{"empty output", "", nil},
		{"truncated output", "1 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\nInst partial [1.0]", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result, err := checkProcursusAPTWith(context.Background(), layout, func(context.Context, string, ...string) (string, error) {
				return tt.output, tt.err
			})
			assert.Error(t, err)
			assert.Nil(t, result.counts, "query failure must not be represented as a verified zero")
			assert.Nil(t, result.packages)
			assert.False(t, result.securityKnown)
		})
	}
}

func TestProcursusCommandAllowlistRejectsMutatingArgv(t *testing.T) {
	allowed := []struct {
		name, executable string
		args             []string
	}{
		{"apt version", "/var/jb/usr/bin/apt-get", []string{"--version"}},
		{"apt config", "/var/jb/usr/bin/apt-config", []string{"dump"}},
		{"dpkg package marker", "/var/jb/usr/bin/dpkg-query", []string{"-W", "-f=${Package} ${Status}\\n", "apt", "procursus-keyring"}},
		{"simulated update listing", "/var/jb/usr/bin/apt-get", []string{"-s", "--no-download", "-o", "Debug::NoLocking=1", "-o", "Dir::Cache::pkgcache=", "-o", "Dir::Cache::srcpkgcache=", "dist-upgrade"}},
	}
	for _, tt := range allowed {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, procursusCommandAllowed(tt.executable, tt.args))
		})
	}
	for _, tt := range []struct {
		name, executable string
		args             []string
	}{
		{"apt repository refresh", "/var/jb/usr/bin/apt-get", []string{"update"}},
		{"apt upgrade", "/var/jb/usr/bin/apt-get", []string{"upgrade"}},
		{"apt dist upgrade without simulation", "/var/jb/usr/bin/apt-get", []string{"dist-upgrade"}},
		{"apt simulation missing no-download", "/var/jb/usr/bin/apt-get", []string{"-s", "dist-upgrade"}},
		{"apt install", "/var/jb/usr/bin/apt-get", []string{"install", "fixture"}},
		{"apt remove", "/var/jb/usr/bin/apt-get", []string{"remove", "fixture"}},
		{"apt autoremove", "/var/jb/usr/bin/apt-get", []string{"autoremove"}},
		{"dpkg install", "/var/jb/usr/bin/dpkg", []string{"-i", "fixture.deb"}},
		{"dpkg configure", "/var/jb/usr/bin/dpkg", []string{"--configure", "-a"}},
		{"dpkg remove", "/var/jb/usr/bin/dpkg", []string{"--remove", "fixture"}},
		{"shell execution", "/bin/sh", []string{"-c", "apt-get update"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, procursusCommandAllowed(tt.executable, tt.args))
		})
	}
}

func TestProcursusCommandBoundsCancellationAndChildCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires executable shell fixtures")
	}
	binDir := t.TempDir()
	writeCommand := func(name, body string, mode os.FileMode) string {
		path := filepath.Join(binDir, name)
		require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), mode))
		return path
	}

	t.Run("missing binary", func(t *testing.T) {
		_, err := runProcursusBoundedCommand(context.Background(), 64, filepath.Join(binDir, "missing"))
		assert.Error(t, err)
	})
	t.Run("permission denied", func(t *testing.T) {
		path := writeCommand("not-executable", "exit 0\n", 0o600)
		_, err := runProcursusBoundedCommand(context.Background(), 64, path)
		assert.Error(t, err)
	})
	t.Run("nonzero exit", func(t *testing.T) {
		path := writeCommand("nonzero", "exit 7\n", 0o700)
		_, err := runProcursusBoundedCommand(context.Background(), 64, path)
		assert.Error(t, err)
	})
	t.Run("oversized output cancels process", func(t *testing.T) {
		path := writeCommand("oversized", "while :; do printf '0123456789'; done\n", 0o700)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := runProcursusBoundedCommand(ctx, 64, path)
		assert.ErrorIs(t, err, errProcursusProbeOutputTooLarge)
	})
	t.Run("timeout kills child retaining stdout", func(t *testing.T) {
		marker := filepath.Join(binDir, "child-survived")
		path := writeCommand("child-pipe", "(/bin/sleep 0.5; /bin/echo leaked > '"+marker+"') &\nwait\n", 0o700)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, err := runProcursusBoundedCommand(ctx, 64, path)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, time.Since(started), 2*time.Second, "WaitDelay should bound inherited pipes")
		time.Sleep(600 * time.Millisecond)
		assert.NoFileExists(t, marker, "a child process outlived the canceled package query")
	})
}

func TestProcursusAPTCountLimit(t *testing.T) {
	var output strings.Builder
	output.WriteString("8193 upgraded, 0 newly installed, 0 to remove and 0 not upgraded.\n")
	for i := 0; i <= procursusPackageLimit; i++ {
		fmt.Fprintf(&output, "Inst package-%05d [1.0] (2.0 repo)\n", i)
	}
	_, err := parseProcursusAPTOutput(output.String())
	assert.ErrorContains(t, err, "count exceeded limit")
}

func TestProcursusAPTQueryReadOnlyAndNetworkIsolatedFixture(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network namespace and read-only root sandbox are validated by the Linux P011 job")
	}
	aptGet, err := exec.LookPath("apt-get")
	if err != nil {
		t.Skip("apt-get is unavailable on this test host")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	var connections atomic.Int32
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connections.Add(1)
			_ = conn.Close()
		}
	}()
	closeListener := func() {
		_ = listener.Close()
		<-acceptDone
	}
	t.Cleanup(closeListener)

	root := t.TempDir()
	for _, dir := range []string{
		"etc/apt/apt.conf.d", "etc/apt/preferences.d", "etc/apt/sources.list.d",
		"var/lib/apt/lists/partial", "var/lib/dpkg/info", "var/lib/dpkg/updates",
		"var/lib/dpkg/triggers", "var/cache/apt/archives/partial",
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
	}
	sourceURL := fmt.Sprintf("http://127.0.0.1:%d/procursus", port)
	require.NoError(t, os.WriteFile(filepath.Join(root, "etc/apt/sources.list"), []byte("deb [trusted=yes] "+sourceURL+" stable main\n"), 0o644))
	status := "Package: held-fixture\nStatus: hold ok installed\nPriority: optional\nSection: misc\nInstalled-Size: 1\nMaintainer: fixture <fixture@example.invalid>\nArchitecture: iphoneos-arm\nVersion: 1.0-1\nDescription: synthetic held package\n\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/status"), []byte(status), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/available"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/status-old"), []byte(status), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/info/held-fixture.list"), []byte("/usr/share/fixture\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/info/format"), []byte("1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/updates/0000"), []byte("pending-state-fixture\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/dpkg/triggers/File"), []byte("/usr/share/fixture held-fixture\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/apt/extended_states"), []byte(""), 0o644))
	indexName := fmt.Sprintf("127.0.0.1:%d_procursus_dists_stable_main_binary-iphoneos-arm_Packages", port)
	require.NoError(t, os.WriteFile(filepath.Join(root, "var/lib/apt/lists", indexName), []byte(""), 0o644))
	aptConfig := strings.Join([]string{
		"Dir " + strconv.Quote(root) + ";",
		"Dir::State " + strconv.Quote(filepath.Join(root, "var/lib/apt")) + ";",
		"Dir::State::status " + strconv.Quote(filepath.Join(root, "var/lib/dpkg/status")) + ";",
		"Dir::State::lists " + strconv.Quote(filepath.Join(root, "var/lib/apt/lists")) + ";",
		"Dir::State::extended_states " + strconv.Quote(filepath.Join(root, "var/lib/apt/extended_states")) + ";",
		"Dir::Cache " + strconv.Quote(filepath.Join(root, "var/cache/apt")) + ";",
		"Dir::Cache::pkgcache \"\";",
		"Dir::Cache::srcpkgcache \"\";",
		"Dir::Cache::archives " + strconv.Quote(filepath.Join(root, "var/cache/apt/archives")) + ";",
		"Dir::Etc " + strconv.Quote(filepath.Join(root, "etc/apt")) + ";",
		"Dir::Etc::sourcelist " + strconv.Quote(filepath.Join(root, "etc/apt/sources.list")) + ";",
		"Dir::Etc::sourceparts " + strconv.Quote(filepath.Join(root, "etc/apt/sources.list.d")) + ";",
		"Dir::Etc::main " + strconv.Quote(filepath.Join(root, "etc/apt/apt.conf")) + ";",
		"Dir::Etc::parts " + strconv.Quote(filepath.Join(root, "etc/apt/apt.conf.d")) + ";",
		"Dir::Etc::preferences " + strconv.Quote(filepath.Join(root, "etc/apt/preferences")) + ";",
		"Dir::Etc::preferencesparts " + strconv.Quote(filepath.Join(root, "etc/apt/preferences.d")) + ";",
		"APT::Architecture \"iphoneos-arm\";",
		"APT::Architectures { \"iphoneos-arm\"; };",
	}, "\n") + "\n"
	configPath := filepath.Join(root, "etc/apt/apt.conf")
	require.NoError(t, os.WriteFile(configPath, []byte(aptConfig), 0o644))

	before := snapshotPackageFixture(t, root)
	sandboxPath, sandboxArgs, isolated := usableOfflineSandbox()
	if !isolated {
		t.Skip("this host has no network-disabled, read-only filesystem sandbox; the required Linux fixture must run in a capable cloud environment")
	}
	t.Log("APT query ran with network namespace disabled and the filesystem read-only")
	argsSeen := false
	result, err := checkProcursusAPTWith(context.Background(), procursusAPTLayout{aptGet: aptGet}, func(ctx context.Context, executable string, args ...string) (string, error) {
		require.Equal(t, aptGet, executable)
		require.Equal(t, procursusAPTReadOnlyArgs, args)
		argsSeen = true
		env := []string{"LC_ALL=C", "APT_CONFIG=" + configPath}
		command := sandboxPath
		commandArgs := append(append(append([]string(nil), sandboxArgs...), executable), args...)
		cmd := exec.CommandContext(ctx, command, commandArgs...)
		cmd.Env = env
		cmd.WaitDelay = 2 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			t.Logf("disposable APT fixture failed; stdout=%q stderr=%q", stdout.String(), stderr.String())
		}
		return stdout.String(), err
	})
	require.NoError(t, err)
	assert.True(t, argsSeen)
	assert.Equal(t, []uint16{0}, result.counts)
	assert.Empty(t, result.packages)
	assert.False(t, result.securityKnown)
	after := snapshotPackageFixture(t, root)
	assert.Equal(t, before, after, "APT simulation changed fixture files, package databases, indexes, caches, or directory metadata")
	t.Logf("fixture entries=%d SHA256 before=%s after=%s", len(before), hashFixtureSnapshot(before), hashFixtureSnapshot(after))
	closeListener()
	assert.Zero(t, connections.Load(), "APT simulation attempted network access")
}

func usableOfflineSandbox() (string, []string, bool) {
	if bwrap, err := exec.LookPath("bwrap"); err == nil {
		args := []string{"--unshare-net", "--ro-bind", "/", "/", "--"}
		cmd := exec.Command(bwrap, append(args, "/bin/true")...)
		cmd.Env = []string{"LC_ALL=C"}
		if cmd.Run() == nil {
			return bwrap, args, true
		}
	}
	return "", nil, false
}

func snapshotPackageFixture(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		digest := "directory"
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			hash := sha256.New()
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			digest = hex.EncodeToString(hash.Sum(nil))
		}
		snapshot[rel] = fmt.Sprintf("%s:%o:%d:%d", digest, info.Mode(), info.Size(), info.ModTime().UnixNano())
		return nil
	})
	require.NoError(t, err)
	return snapshot
}

func hashFixtureSnapshot(snapshot map[string]string) string {
	keys := make([]string, 0, len(snapshot))
	for key := range snapshot {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		_, _ = fmt.Fprintf(hash, "%s\x00%s\n", key, snapshot[key])
	}
	return hex.EncodeToString(hash.Sum(nil))
}
