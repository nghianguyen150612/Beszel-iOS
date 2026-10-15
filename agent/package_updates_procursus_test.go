//go:build testing

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
			"Dir::State::status \"/var/lib/dpkg/status\";\n" +
			"Dir::State::lists \"lists/\";\n" +
			"Dir::Etc \"etc/apt\";\n" +
			"Dir::Etc::sourcelist \"sources.list\";\n" +
			"Dir::Etc::sourceparts \"sources.list.d\";\n",
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
