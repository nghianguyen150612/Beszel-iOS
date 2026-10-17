//go:build testing

package site

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEmbeddedFrontendMatchesManifest(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate embed test source")
	}
	manifestPath := filepath.Join(filepath.Dir(sourceFile), "..", "..", "docs", "ios-frontend-assets-manifest.sha256")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read frontend asset manifest: %v", err)
	}
	want := make(map[string]string)
	for lineNumber, line := range strings.Split(strings.TrimSpace(string(manifestBytes)), "\n") {
		digest, name, found := strings.Cut(line, "  ")
		if !found || len(digest) != 64 || name == "" {
			t.Fatalf("invalid manifest line %d: %q", lineNumber+1, line)
		}
		if _, exists := want[name]; exists {
			t.Fatalf("duplicate manifest asset %q", name)
		}
		want[name] = digest
	}

	got := make(map[string]string)
	err = fs.WalkDir(DistDirFS, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		content, err := fs.ReadFile(DistDirFS, name)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(content)
		got[name] = fmt.Sprintf("%x", digest)
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded frontend: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("embedded frontend is empty")
	}
	if len(got) != len(want) {
		t.Fatalf("embedded asset count=%d, manifest count=%d", len(got), len(want))
	}
	for name, expectedDigest := range want {
		if digest, ok := got[name]; !ok {
			t.Errorf("manifest asset %q is not embedded", name)
		} else if digest != expectedDigest {
			t.Errorf("embedded asset %q hash=%s, want %s", name, digest, expectedDigest)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("embedded asset %q is absent from the manifest", name)
		}
	}
	for _, required := range []string{"index.html"} {
		if _, ok := got[required]; !ok {
			t.Errorf("required embedded asset %q is missing", required)
		}
	}
	if !hasAssetSuffix(got, ".js") || !hasAssetSuffix(got, ".css") {
		t.Fatal("embedded frontend is missing a JavaScript or CSS asset")
	}
}

func hasAssetSuffix(assets map[string]string, suffix string) bool {
	for name := range assets {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}
