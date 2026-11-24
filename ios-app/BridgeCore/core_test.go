package bridgecore

import (
	"runtime"
	"testing"

	"github.com/henrygd/beszel"
)

func TestVersionIsSharedBeszelVersion(t *testing.T) {
	if got := Version(); got != beszel.Version {
		t.Fatalf("bridge version = %q, shared version = %q", got, beszel.Version)
	}
}

func TestPlatformMatchesGoRuntime(t *testing.T) {
	if got, want := Platform(), runtime.GOOS+"/"+runtime.GOARCH; got != want {
		t.Fatalf("bridge platform = %q, want %q", got, want)
	}
}
