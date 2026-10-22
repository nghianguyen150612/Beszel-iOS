// check-openpgp-imports fails if OpenPGP packages are directly imported by any
// repository Go source or appear in the selected Agent/Hub dependency graphs.
// The syntax scan sees build-tagged and test files regardless of the host OS.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Import struct {
	Path string `json:"path"`
	File string `json:"file"`
}

type Profile struct {
	Name            string   `json:"name"`
	PackageCount    int      `json:"package_count"`
	OpenPGPPackages []string `json:"openpgp_packages"`
}

type Evidence struct {
	Schema             int       `json:"schema"`
	Status             string    `json:"status"`
	SourceFilesScanned int       `json:"source_files_scanned"`
	SourceImports      []Import  `json:"source_imports"`
	Profiles           []Profile `json:"dependency_profiles,omitempty"`
	Errors             []string  `json:"errors,omitempty"`
}

type profileSpec struct {
	name string
	args []string
	env  map[string]string
}

func main() {
	rootFlag := flag.String("root", ".", "repository root")
	outputFlag := flag.String("output", "", "optional JSON evidence path")
	skipDeps := flag.Bool("skip-deps", false, "only scan source imports; intended for unit tests")
	flag.Parse()

	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		fatal(err)
	}
	evidence := Evidence{Schema: 1, Status: "PASS", SourceImports: []Import{}}
	evidence.SourceImports, evidence.SourceFilesScanned, err = scanSourceImports(root)
	if err != nil {
		evidence.Errors = append(evidence.Errors, err.Error())
	}
	if !*skipDeps && err == nil {
		evidence.Profiles, err = scanDependencyProfiles(root)
		if err != nil {
			evidence.Errors = append(evidence.Errors, err.Error())
		}
	}
	if len(evidence.SourceImports) > 0 {
		evidence.Errors = append(evidence.Errors, "repository source imports a golang.org/x/crypto/openpgp package")
	}
	for _, profile := range evidence.Profiles {
		if len(profile.OpenPGPPackages) > 0 {
			evidence.Errors = append(evidence.Errors, "dependency profile "+profile.Name+" includes an OpenPGP package")
		}
	}
	sort.Slice(evidence.SourceImports, func(i, j int) bool {
		if evidence.SourceImports[i].File == evidence.SourceImports[j].File {
			return evidence.SourceImports[i].Path < evidence.SourceImports[j].Path
		}
		return evidence.SourceImports[i].File < evidence.SourceImports[j].File
	})
	if len(evidence.Errors) > 0 {
		evidence.Status = "FAIL"
	}
	rendered, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		fatal(err)
	}
	rendered = append(rendered, '\n')
	fmt.Print(string(rendered))
	if *outputFlag != "" {
		if err := os.WriteFile(*outputFlag, rendered, 0o600); err != nil {
			fatal(err)
		}
	}
	if evidence.Status != "PASS" {
		os.Exit(1)
	}
}

func scanSourceImports(root string) ([]Import, int, error) {
	imports := []Import{}
	files := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		files++
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse Go imports in %s: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("decode import in %s: %w", path, err)
			}
			if strings.HasPrefix(importPath, "golang.org/x/crypto/openpgp") {
				relative, _ := filepath.Rel(root, path)
				imports = append(imports, Import{Path: importPath, File: filepath.ToSlash(relative)})
			}
		}
		return nil
	})
	return imports, files, err
}

func scanDependencyProfiles(root string) ([]Profile, error) {
	profiles := []profileSpec{
		{name: "host-default-with-tests", args: []string{"-deps", "-test", "-json", "./..."}},
		{name: "host-testing-no_ui-with-tests", args: []string{"-deps", "-test", "-json", "-tags=testing,no_ui", "./..."}},
		{name: "ios-arm64-testing-no_ui-with-tests", args: []string{"-deps", "-test", "-json", "-tags=testing,no_ui", "./..."}, env: map[string]string{"GOOS": "ios", "GOARCH": "arm64", "GOARM64": "v8.0", "CGO_ENABLED": "1", "CC": "clang"}},
	}
	results := make([]Profile, 0, len(profiles))
	for _, profile := range profiles {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		cmd := exec.CommandContext(ctx, "go", append([]string{"list"}, profile.args...)...)
		cmd.Dir = root
		cmd.Env = withEnv(os.Environ(), map[string]string{"GOTOOLCHAIN": "local"})
		if len(profile.env) > 0 {
			cmd.Env = withEnv(cmd.Env, profile.env)
		}
		out, err := cmd.Output()
		cancel()
		if err != nil {
			detail := ""
			if exitErr, ok := err.(*exec.ExitError); ok {
				detail = strings.TrimSpace(string(exitErr.Stderr))
			}
			return results, fmt.Errorf("go list profile %s failed: %w: %s", profile.name, err, detail)
		}
		decoder := json.NewDecoder(bytes.NewReader(out))
		result := Profile{Name: profile.name, OpenPGPPackages: []string{}}
		for {
			var pkg struct {
				ImportPath string
			}
			if err := decoder.Decode(&pkg); err != nil {
				if err == io.EOF {
					break
				}
				return results, fmt.Errorf("decode go list profile %s: %w", profile.name, err)
			}
			if pkg.ImportPath == "" {
				continue
			}
			result.PackageCount++
			if strings.HasPrefix(pkg.ImportPath, "golang.org/x/crypto/openpgp") {
				result.OpenPGPPackages = append(result.OpenPGPPackages, pkg.ImportPath)
			}
		}
		sort.Strings(result.OpenPGPPackages)
		results = append(results, result)
	}
	return results, nil
}

func withEnv(source []string, overrides map[string]string) []string {
	result := make([]string, 0, len(source)+len(overrides))
	for _, item := range source {
		key, _, found := strings.Cut(item, "=")
		if _, replaced := overrides[key]; !found || !replaced {
			result = append(result, item)
		}
	}
	for key, value := range overrides {
		result = append(result, key+"="+value)
	}
	return result
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(2)
}
