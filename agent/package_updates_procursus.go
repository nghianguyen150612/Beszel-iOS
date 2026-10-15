package agent

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
)

const (
	procursusProbeOutputLimit = 256 << 10
	procursusQueryOutputLimit = 1 << 20
	procursusPackageLimit     = 8192
	iosPackageUpdatesTimeout  = 20 * time.Second
	procursusPackageIndexMark = "_Packages"
)

var errProcursusProbeOutputTooLarge = errors.New("package-manager probe output exceeded limit")
var errProcursusCommandNotAllowed = errors.New("package-manager command is not allowlisted")

var procursusSummaryPattern = regexp.MustCompile(`^[0-9]+ upgraded, [0-9]+ newly installed, [0-9]+ to remove and [0-9]+ not upgraded\.$`)

type procursusBoundedOutput struct {
	buffer   []byte
	limit    int
	cancel   context.CancelFunc
	overflow bool
}

func (w *procursusBoundedOutput) Write(p []byte) (int, error) {
	remaining := w.limit - len(w.buffer)
	if remaining < len(p) {
		if remaining > 0 {
			w.buffer = append(w.buffer, p[:remaining]...)
		}
		w.overflow = true
		w.cancel()
		return remaining, errProcursusProbeOutputTooLarge
	}
	w.buffer = append(w.buffer, p...)
	return len(p), nil
}

func (w *procursusBoundedOutput) String() string { return string(w.buffer) }

func runProcursusProbeCommand(ctx context.Context, executable string, args ...string) (string, error) {
	return runProcursusBoundedCommand(ctx, procursusProbeOutputLimit, executable, args...)
}

func procursusCommandAllowed(executable string, args []string) bool {
	switch filepath.Base(executable) {
	case "apt-get":
		return slices.Equal(args, []string{"--version"}) || slices.Equal(args, procursusAPTReadOnlyArgs)
	case "apt-config":
		return slices.Equal(args, []string{"dump"})
	case "dpkg-query":
		return slices.Equal(args, []string{"-W", "-f=${Package} ${Status}\\n", "apt", "procursus-keyring"})
	default:
		return false
	}
}

func runProcursusBoundedCommand(ctx context.Context, limit int, executable string, args ...string) (string, error) {
	return runProcursusBoundedCommandEnv(ctx, limit, []string{"LC_ALL=C"}, executable, args...)
}

func runProcursusBoundedCommandEnv(ctx context.Context, limit int, env []string, executable string, args ...string) (string, error) {
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, executable, args...)
	cmd.Env = append([]string(nil), env...)
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 2 * time.Second
	configureProcursusProcess(cmd)
	output := &procursusBoundedOutput{limit: limit, cancel: cancel}
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
	kind           string
	binDir         string
	aptGet         string
	aptConfig      string
	dpkgQuery      string
	statusPath     string
	listsPath      string
	aptEtcPath     string
	aptConfigFile  string
	aptConfigParts string
	sourceList     string
	sourceParts    string
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
	mainConfig := config["Dir::Etc::main"]
	configParts := config["Dir::Etc::parts"]
	if status == "" || lists == "" || sourceList == "" || sourceParts == "" || mainConfig == "" || configParts == "" {
		return procursusAPTLayout{}, errors.New("missing apt state configuration")
	}
	statePath := resolveAPTConfigPath(root, "", state)
	etcPath := resolveAPTConfigPath(root, "", etc)
	layout := procursusAPTLayout{
		kind:           candidate.kind,
		binDir:         candidate.binDir,
		aptGet:         candidate.aptGet,
		aptConfig:      candidate.aptConfig,
		dpkgQuery:      candidate.dpkgQuery,
		statusPath:     resolveAPTConfigPath(root, statePath, status),
		listsPath:      resolveAPTConfigPath(root, statePath, lists),
		aptEtcPath:     etcPath,
		aptConfigFile:  resolveAPTConfigPath(root, etcPath, mainConfig),
		aptConfigParts: resolveAPTConfigPath(root, etcPath, configParts),
		sourceList:     resolveAPTConfigPath(root, etcPath, sourceList),
		sourceParts:    resolveAPTConfigPath(root, etcPath, sourceParts),
	}
	return layout, nil
}

func resolveAPTConfigPath(root, base, configured string) string {
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
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
	for _, path := range []string{layout.aptEtcPath, layout.aptConfigParts, layout.sourceParts} {
		info, err := probe.stat(path)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return errors.New("APT configuration directory is not a directory")
		}
	}
	if info, err := probe.stat(layout.aptConfigFile); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil && !info.Mode().IsRegular() {
		return errors.New("apt main configuration path is not a file")
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

// newProcursusAPTCheck reads only already-present APT state. The simulation flag
// disables locking and package actions; no-download and empty cache destinations
// make the query fail closed if APT tries to acquire packages or persist caches.
func newProcursusAPTCheck(layout procursusAPTLayout) packageUpdatesCheck {
	return func(ctx context.Context) (packageUpdatesResult, error) {
		return checkProcursusAPTWith(ctx, layout, runProcursusAPTCommand)
	}
}

type procursusAPTCommand func(context.Context, string, ...string) (string, error)

var procursusAPTReadOnlyArgs = []string{
	"-s",
	"--no-download",
	"-o", "Debug::NoLocking=1",
	"-o", "Dir::Cache::pkgcache=",
	"-o", "Dir::Cache::srcpkgcache=",
	"dist-upgrade",
}

func runProcursusAPTCommand(ctx context.Context, executable string, args ...string) (string, error) {
	if filepath.Base(executable) != "apt-get" || !slices.Equal(args, procursusAPTReadOnlyArgs) {
		return "", errProcursusCommandNotAllowed
	}
	return runProcursusBoundedCommand(ctx, procursusQueryOutputLimit, executable, args...)
}

func checkProcursusAPTWith(ctx context.Context, layout procursusAPTLayout, run procursusAPTCommand) (packageUpdatesResult, error) {
	queryCtx, cancel := context.WithTimeout(ctx, iosPackageUpdatesTimeout)
	defer cancel()
	output, err := run(queryCtx, layout.aptGet, procursusAPTReadOnlyArgs...)
	if err != nil {
		return packageUpdatesResult{}, err
	}
	packages, err := parseProcursusAPTOutput(output)
	if err != nil {
		return packageUpdatesResult{}, err
	}
	return packageUpdatesResult{
		counts:        []uint16{uint16(len(packages))},
		packages:      packages,
		securityKnown: false,
	}, nil
}

// parseProcursusAPTOutput accepts complete, C-locale apt-get simulation output.
// New dependencies, removals, held packages and downgrades are not updates to an
// installed package. Security classification is deliberately not inferred from
// repository text.
func parseProcursusAPTOutput(output string) ([]system.PackageUpdate, error) {
	if len(output) > procursusQueryOutputLimit {
		return nil, errProcursusProbeOutputTooLarge
	}
	seenSummary := false
	var summaryUpgrades uint64
	seen := make(map[string]system.PackageUpdate)
	seenInst := make(map[string]system.PackageUpdate)
	downgrades := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 4096), 64<<10)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if procursusSummaryPattern.MatchString(line) {
			if seenSummary {
				return nil, errors.New("duplicate APT simulation summary")
			}
			fields := strings.Fields(line)
			upgrades, err := strconv.ParseUint(fields[0], 10, 64)
			if err != nil {
				return nil, errors.New("invalid APT simulation summary count")
			}
			summaryUpgrades = upgrades
			seenSummary = true
			continue
		}
		if !strings.HasPrefix(line, "Inst ") {
			continue
		}
		pkg, hasCurrent, err := parseProcursusAPTInst(line)
		if err != nil {
			return nil, err
		}
		if prior, ok := seenInst[pkg.Name]; ok {
			if prior != pkg {
				return nil, errors.New("conflicting duplicate APT package entry")
			}
			continue
		}
		if len(seenInst) >= procursusPackageLimit {
			return nil, errors.New("APT simulation package count exceeded limit")
		}
		seenInst[pkg.Name] = pkg
		if !hasCurrent {
			continue
		}
		comparison, err := compareDpkgVersions(pkg.Available, pkg.Current)
		if err != nil {
			return nil, err
		}
		if comparison <= 0 {
			if comparison < 0 {
				downgrades[pkg.Name] = struct{}{}
			}
			continue
		}
		seen[pkg.Name] = pkg
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New("APT simulation output is malformed or truncated")
	}
	if !seenSummary {
		return nil, errors.New("APT simulation summary is missing")
	}
	updateCount := uint64(len(seen))
	if summaryUpgrades != updateCount && summaryUpgrades != updateCount+uint64(len(downgrades)) {
		return nil, errors.New("APT simulation summary does not match parsed package entries")
	}
	packages := make([]system.PackageUpdate, 0, len(seen))
	for _, pkg := range seen {
		packages = append(packages, pkg)
	}
	slices.SortFunc(packages, func(a, b system.PackageUpdate) int {
		if n := strings.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		if n := strings.Compare(a.Current, b.Current); n != 0 {
			return n
		}
		return strings.Compare(a.Available, b.Available)
	})
	return packages, nil
}

func parseProcursusAPTInst(line string) (system.PackageUpdate, bool, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "Inst "))
	name, rest, ok := strings.Cut(rest, " ")
	if !ok || !validAPTQualifiedPackageName(name) {
		return system.PackageUpdate{}, false, errors.New("malformed APT simulation package name")
	}
	rest = strings.TrimSpace(rest)
	current := ""
	if strings.HasPrefix(rest, "[") {
		end := strings.IndexByte(rest, ']')
		if end <= 1 {
			return system.PackageUpdate{}, false, errors.New("malformed APT installed version")
		}
		current = rest[1:end]
		rest = strings.TrimSpace(rest[end+1:])
	} else if !strings.HasPrefix(rest, "(") {
		return system.PackageUpdate{}, false, errors.New("malformed APT simulation installed version")
	}
	open := strings.IndexByte(rest, '(')
	if open < 0 {
		return system.PackageUpdate{}, false, errors.New("APT candidate version is missing")
	}
	afterOpen := strings.TrimSpace(rest[open+1:])
	fields := strings.Fields(afterOpen)
	close := strings.IndexByte(afterOpen, ')')
	if len(fields) == 0 || close < len(fields[0]) || !validDpkgVersion(fields[0]) {
		return system.PackageUpdate{}, false, errors.New("malformed APT candidate version")
	}
	pkg := system.PackageUpdate{Name: name, Current: current, Available: fields[0]}
	if current == "" {
		return pkg, false, nil
	}
	if !validDpkgVersion(current) {
		return system.PackageUpdate{}, false, errors.New("malformed APT installed version")
	}
	return pkg, true, nil
}

func validAPTQualifiedPackageName(name string) bool {
	parts := strings.Split(name, ":")
	if len(parts) > 2 || !validAPTNamePart(parts[0], true) {
		return false
	}
	return len(parts) == 1 || validAPTNamePart(parts[1], false)
}

func validAPTNamePart(s string, packageName bool) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		valid := c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '.' || c == '-'
		if !valid || (i == 0 && packageName && !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func validDpkgVersion(version string) bool {
	if version == "" || strings.ContainsAny(version, " \t\r\n\x00") {
		return false
	}
	epoch, rest := splitDpkgEpoch(version)
	if epoch == "" || rest == "" {
		return false
	}
	for _, c := range epoch {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func splitDpkgEpoch(version string) (epoch, rest string) {
	colon := strings.IndexByte(version, ':')
	if colon < 0 {
		return "0", version
	}
	if colon == 0 {
		return "", version
	}
	for _, c := range version[:colon] {
		if c < '0' || c > '9' {
			return "", version
		}
	}
	epoch = strings.TrimLeft(version[:colon], "0")
	if epoch == "" {
		epoch = "0"
	}
	return epoch, version[colon+1:]
}

// compareDpkgVersions implements dpkg's epoch and verrevcmp ordering for the
// version strings present in apt-get simulation lines.
func compareDpkgVersions(a, b string) (int, error) {
	if !validDpkgVersion(a) || !validDpkgVersion(b) {
		return 0, errors.New("invalid Debian package version")
	}
	aEpoch, aRest := splitDpkgEpoch(a)
	bEpoch, bRest := splitDpkgEpoch(b)
	if n := compareDpkgNumeric(aEpoch, bEpoch); n != 0 {
		return n, nil
	}
	aUpstream, aRevision := splitDpkgRevision(aRest)
	bUpstream, bRevision := splitDpkgRevision(bRest)
	if n := dpkgRevisionCompare(aUpstream, bUpstream); n != 0 {
		return n, nil
	}
	return dpkgRevisionCompare(aRevision, bRevision), nil
}

func splitDpkgRevision(version string) (upstream, revision string) {
	if dash := strings.LastIndexByte(version, '-'); dash >= 0 {
		return version[:dash], version[dash+1:]
	}
	return version, "0"
}

func compareDpkgNumeric(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return strings.Compare(a, b)
}

func dpkgRevisionCompare(a, b string) int {
	ai, bi := 0, 0
	for {
		for {
			var ac, bc byte
			if ai < len(a) {
				ac = a[ai]
			}
			if bi < len(b) {
				bc = b[bi]
			}
			if isASCIIDigit(ac) || isASCIIDigit(bc) {
				break
			}
			if order := dpkgCharOrder(ac) - dpkgCharOrder(bc); order != 0 {
				if order < 0 {
					return -1
				}
				return 1
			}
			if ac == 0 {
				return 0
			}
			ai++
			bi++
		}
		for ai < len(a) && a[ai] == '0' {
			ai++
		}
		for bi < len(b) && b[bi] == '0' {
			bi++
		}
		for {
			aDigit := ai < len(a) && isASCIIDigit(a[ai])
			bDigit := bi < len(b) && isASCIIDigit(b[bi])
			if !aDigit && !bDigit {
				break
			}
			if !aDigit {
				return -1
			}
			if !bDigit {
				return 1
			}
			if a[ai] != b[bi] {
				if a[ai] < b[bi] {
					return -1
				}
				return 1
			}
			ai++
			bi++
		}
	}
}

func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

func dpkgCharOrder(c byte) int {
	switch {
	case c == '~':
		return -1
	case c == 0:
		return 0
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		return int(c)
	default:
		return int(c) + 256
	}
}

var _ io.Closer = (procursusReadCloser)(nil)
