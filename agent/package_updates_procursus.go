package agent

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	procursusProbeOutputLimit = 256 << 10
	procursusPackageIndexMark = "_Packages"
)

var errProcursusProbeOutputTooLarge = errors.New("package-manager probe output exceeded limit")

type procursusBoundedOutput struct {
	bytes.Buffer
	limit    int
	cancel   context.CancelFunc
	overflow bool
}

func (w *procursusBoundedOutput) Write(p []byte) (int, error) {
	remaining := w.limit - w.Len()
	if remaining < len(p) {
		if remaining > 0 {
			_, _ = w.Buffer.Write(p[:remaining])
		}
		w.overflow = true
		w.cancel()
		return remaining, errProcursusProbeOutputTooLarge
	}
	return w.Buffer.Write(p)
}

func runProcursusProbeCommand(ctx context.Context, executable string, args ...string) (string, error) {
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, executable, args...)
	cmd.Env = []string{"LC_ALL=C"}
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 2 * time.Second
	output := &procursusBoundedOutput{limit: procursusProbeOutputLimit, cancel: cancel}
	cmd.Stdout = output
	err := cmd.Run()
	if output.overflow {
		return "", errProcursusProbeOutputTooLarge
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		return "", err
	}
	return output.String(), nil
}

type procursusDetectionStatus uint8

const (
	procursusNoManager procursusDetectionStatus = iota
	procursusSupported
	procursusIncomplete
	procursusUnsupported
	procursusAmbiguous
	procursusInsufficientPermission
)

func (s procursusDetectionStatus) String() string {
	switch s {
	case procursusSupported:
		return "supported"
	case procursusIncomplete:
		return "incomplete"
	case procursusUnsupported:
		return "unsupported"
	case procursusAmbiguous:
		return "ambiguous"
	case procursusInsufficientPermission:
		return "insufficient-permission"
	default:
		return "no-package-manager"
	}
}

type procursusReadCloser interface{ Close() error }

// procursusProbe isolates path and subprocess access so detection can be tested
// against disposable rootful/rootless trees without consulting the host.
type procursusProbe interface {
	stat(string) (os.FileInfo, error)
	readDir(string) ([]os.DirEntry, error)
	open(string) (procursusReadCloser, error)
	evalSymlinks(string) (string, error)
	lookPath(string) (string, error)
	run(context.Context, string, ...string) (string, error)
}

type procursusAPTLayout struct {
	kind        string
	binDir      string
	aptGet      string
	aptConfig   string
	dpkgQuery   string
	statusPath  string
	listsPath   string
	aptEtcPath  string
	sourceList  string
	sourceParts string
}

type procursusDetection struct {
	status procursusDetectionStatus
	layout procursusAPTLayout
}

type procursusCandidate struct {
	kind       string
	binDir     string
	aptGet     string
	aptConfig  string
	dpkgQuery  string
	any        bool
	complete   bool
	permission bool
	aptGetReal string
}

func detectProcursusAPT(ctx context.Context, probe procursusProbe, binaryDirs []string) procursusDetection {
	candidates := make([]procursusCandidate, 0, len(binaryDirs))
	anyCandidate := false
	permissionFailure := false
	for _, dir := range binaryDirs {
		candidate := procursusCandidate{
			kind:      procursusLayoutKind(dir),
			binDir:    filepath.Clean(dir),
			aptGet:    filepath.Join(dir, "apt-get"),
			aptConfig: filepath.Join(dir, "apt-config"),
			dpkgQuery: filepath.Join(dir, "dpkg-query"),
		}
		for _, path := range []string{candidate.aptGet, candidate.aptConfig, candidate.dpkgQuery} {
			info, err := probe.stat(path)
			if err != nil {
				if os.IsPermission(err) {
					candidate.permission = true
				}
				continue
			}
			candidate.any = true
			if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
				candidate.permission = true
			}
		}
		candidate.complete = candidate.any && executableFile(probe, candidate.aptGet) && executableFile(probe, candidate.aptConfig) && executableFile(probe, candidate.dpkgQuery)
		if candidate.any {
			anyCandidate = true
		}
		if candidate.permission {
			permissionFailure = true
		}
		if candidate.complete {
			candidate.aptGetReal = resolvedPath(probe, candidate.aptGet)
		}
		candidates = append(candidates, candidate)
	}

	var complete []procursusCandidate
	for _, candidate := range candidates {
		if candidate.complete {
			complete = append(complete, candidate)
		}
	}
	if len(complete) > 1 {
		for i := 1; i < len(complete); i++ {
			if complete[0].aptGetReal != complete[i].aptGetReal {
				return procursusDetection{status: procursusAmbiguous}
			}
		}
		complete = complete[:1]
	}
	if len(complete) == 0 {
		if path, err := probe.lookPath("apt-get"); err == nil && !inDirectories(path, binaryDirs) {
			return procursusDetection{status: procursusUnsupported}
		}
		if permissionFailure {
			return procursusDetection{status: procursusInsufficientPermission}
		}
		if anyCandidate {
			return procursusDetection{status: procursusIncomplete}
		}
		return procursusDetection{status: procursusNoManager}
	}
	selected := complete[0]
	for _, candidate := range candidates {
		if candidate.any && candidate.aptGet != selected.aptGet && candidate.aptGetReal != selected.aptGetReal {
			if executableFile(probe, candidate.aptGet) {
				return procursusDetection{status: procursusAmbiguous}
			}
		}
	}

	version, err := probe.run(ctx, selected.aptGet, "--version")
	if err != nil || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(version)), "apt ") {
		return procursusDetection{status: procursusUnsupported}
	}
	configOutput, err := probe.run(ctx, selected.aptConfig, "dump")
	if err != nil {
		if os.IsPermission(err) {
			return procursusDetection{status: procursusInsufficientPermission}
		}
		return procursusDetection{status: procursusIncomplete}
	}
	config, err := parseProcursusAPTConfig(configOutput)
	if err != nil {
		return procursusDetection{status: procursusIncomplete}
	}
	layout, err := resolveProcursusAPTLayout(selected, config)
	if err != nil {
		return procursusDetection{status: procursusUnsupported}
	}
	if err := validateProcursusAPTPaths(probe, layout); err != nil {
		if os.IsPermission(err) {
			return procursusDetection{status: procursusInsufficientPermission}
		}
		return procursusDetection{status: procursusIncomplete}
	}
	packages, err := probe.run(ctx, selected.dpkgQuery, "-W", "-f=${Package} ${Status}\\n", "apt", "procursus-keyring")
	if err != nil {
		if os.IsPermission(err) {
			return procursusDetection{status: procursusInsufficientPermission}
		}
		return procursusDetection{status: procursusUnsupported}
	}
	if !installedProcursusPackages(packages) {
		return procursusDetection{status: procursusUnsupported}
	}
	return procursusDetection{status: procursusSupported, layout: layout}
}

func executableFile(probe procursusProbe, path string) bool {
	info, err := probe.stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

func resolvedPath(probe procursusProbe, path string) string {
	resolved, err := probe.evalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(path)
}

func procursusLayoutKind(binDir string) string {
	if filepath.Clean(binDir) == "/var/jb/usr/bin" || strings.HasSuffix(filepath.Clean(binDir), string(filepath.Separator)+"var"+string(filepath.Separator)+"jb"+string(filepath.Separator)+"usr"+string(filepath.Separator)+"bin") {
		return "rootless"
	}
	return "rootful"
}

func inDirectories(path string, dirs []string) bool {
	path = filepath.Clean(path)
	for _, dir := range dirs {
		if path == filepath.Join(filepath.Clean(dir), "apt-get") {
			return true
		}
	}
	return false
}

func parseProcursusAPTConfig(output string) (map[string]string, error) {
	values := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 1024), procursusProbeOutputLimit)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, " ")
		if !ok || !strings.HasPrefix(key, "Dir") {
			continue
		}
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(value), ";"))
		if value == "" {
			return nil, errors.New("empty apt directory setting")
		}
		if strings.HasPrefix(value, `"`) {
			decoded, err := strconv.Unquote(value)
			if err != nil {
				return nil, errors.New("invalid apt directory setting")
			}
			value = decoded
		}
		if strings.ContainsAny(value, "\\\x00\n\r") {
			return nil, errors.New("unsafe apt directory setting")
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func resolveProcursusAPTLayout(candidate procursusCandidate, config map[string]string) (procursusAPTLayout, error) {
	root := config["Dir"]
	if root == "" {
		root = "/"
	}
	if !filepath.IsAbs(root) || strings.ContainsAny(root, "\\\x00\n\r") {
		return procursusAPTLayout{}, errors.New("invalid apt root")
	}
	root = filepath.Clean(root)
	state := config["Dir::State"]
	if state == "" {
		return procursusAPTLayout{}, errors.New("missing apt state directory")
	}
	etc := config["Dir::Etc"]
	if etc == "" {
		return procursusAPTLayout{}, errors.New("missing apt configuration directory")
	}
	status := config["Dir::State::status"]
	lists := config["Dir::State::lists"]
	sourceList := config["Dir::Etc::sourcelist"]
	sourceParts := config["Dir::Etc::sourceparts"]
	if status == "" || lists == "" || sourceList == "" || sourceParts == "" {
		return procursusAPTLayout{}, errors.New("missing apt state configuration")
	}
	statePath := resolveAPTConfigPath(root, "", state)
	etcPath := resolveAPTConfigPath(root, "", etc)
	layout := procursusAPTLayout{
		kind:        candidate.kind,
		binDir:      candidate.binDir,
		aptGet:      candidate.aptGet,
		aptConfig:   candidate.aptConfig,
		dpkgQuery:   candidate.dpkgQuery,
		statusPath:  resolveAPTConfigPath(root, statePath, status),
		listsPath:   resolveAPTConfigPath(root, statePath, lists),
		aptEtcPath:  etcPath,
		sourceList:  resolveAPTConfigPath(root, etcPath, sourceList),
		sourceParts: resolveAPTConfigPath(root, etcPath, sourceParts),
	}
	return layout, nil
}

func resolveAPTConfigPath(root, base, configured string) string {
	if filepath.IsAbs(configured) {
		return filepath.Clean(filepath.Join(root, strings.TrimLeft(configured, string(filepath.Separator))))
	}
	if base == "" {
		return filepath.Clean(filepath.Join(root, configured))
	}
	return filepath.Clean(filepath.Join(base, configured))
}

func validateProcursusAPTPaths(probe procursusProbe, layout procursusAPTLayout) error {
	statusSuffix := filepath.Join("var", "lib", "dpkg", "status")
	listsSuffix := filepath.Join("var", "lib", "apt", "lists")
	etcSuffix := filepath.Join("etc", "apt")
	statusPrefix := strings.TrimSuffix(layout.statusPath, statusSuffix)
	if statusPrefix == layout.statusPath || filepath.Clean(statusPrefix) == "." || filepath.Clean(statusPrefix) != filepath.Clean(strings.TrimSuffix(layout.listsPath, listsSuffix)) || filepath.Clean(statusPrefix) != filepath.Clean(strings.TrimSuffix(layout.aptEtcPath, etcSuffix)) {
		return errors.New("apt configuration resolves to inconsistent package paths")
	}
	statusInfo, err := probe.stat(layout.statusPath)
	if err != nil {
		return err
	}
	if !statusInfo.Mode().IsRegular() {
		return errors.New("dpkg status path is not a file")
	}
	file, err := probe.open(layout.statusPath)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	entries, err := probe.readDir(layout.listsPath)
	if err != nil {
		return err
	}
	hasPackageIndex := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), procursusPackageIndexMark) {
			hasPackageIndex = true
			break
		}
	}
	if !hasPackageIndex {
		return errors.New("no cached apt package indexes")
	}
	if _, err := probe.stat(layout.aptEtcPath); err != nil {
		return err
	}
	if _, err := probe.stat(layout.sourceParts); err != nil {
		return err
	}
	if info, err := probe.stat(layout.sourceList); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil && !info.Mode().IsRegular() {
		return errors.New("apt source list is not a file")
	}
	return nil
}

func installedProcursusPackages(output string) bool {
	installed := make(map[string]bool, 2)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 4 || fields[1] != "install" || fields[2] != "ok" || fields[3] != "installed" {
			continue
		}
		if fields[0] == "apt" || fields[0] == "procursus-keyring" {
			installed[fields[0]] = true
		}
	}
	return scanner.Err() == nil && installed["apt"] && installed["procursus-keyring"]
}

// newProcursusAPTCheck is wired in by the iOS factory after a bounded probe.
func newProcursusAPTCheck(layout procursusAPTLayout) packageUpdatesCheck {
	return func(context.Context) (packageUpdatesResult, error) {
		return packageUpdatesResult{}, fmt.Errorf("Procursus query is not initialized")
	}
}

var _ io.Closer = (procursusReadCloser)(nil)
