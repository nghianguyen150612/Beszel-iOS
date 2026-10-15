// Test executable only. Never bundled into Agent, diagnostics or release assets.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const valid = `{"version":1,"source":"mobilewifi-ios12","status":"ok","devices":[{"interface":"en0","powered":true,"station":true,"association":"associated","current_network":true,"ssid_b64":"aG9tZQ==","rssi":{"RSSI_CTL_AGR":-40}}]}`

func hold() {
	for {
		time.Sleep(time.Hour)
	}
}
func main() {
	executable, _ := os.Executable()
	if len(os.Args) == 2 && os.Args[1] == "--hold-stdout" {
		_ = os.WriteFile(executable+".desc.pid", []byte(strconv.Itoa(os.Getpid())), 0600)
		hold()
	}
	if len(os.Args) != 2 || os.Args[1] != "--snapshot-v1" {
		os.Exit(64)
	}
	if os.Getenv("SECRET_SENTINEL") != "" || os.Getenv("DYLD_INSERT_LIBRARIES") != "" {
		os.Exit(65)
	}
	_ = os.WriteFile(executable+".pid", []byte(strconv.Itoa(os.Getpid())), 0600)
	f, err := os.OpenFile(executable+".starts", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err == nil {
		_, _ = fmt.Fprintln(f, os.Getpid())
		_ = f.Close()
	}
	mode := filepath.Base(os.Args[0])
	if mode == "control" {
		if b, err := os.ReadFile(executable + ".mode"); err == nil {
			mode = strings.TrimSpace(string(b))
		}
	}
	switch mode {
	case "valid", "control":
		fmt.Println(valid)
	case "empty":
	case "invalid":
		fmt.Println("{broken")
	case "oversize":
		_, _ = os.Stdout.Write([]byte(strings.Repeat("x", 128*1024)))
		hold()
	case "nonzero":
		os.Exit(23)
	case "crash":
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
		hold()
	case "timeout":
		hold()
	case "unavailable":
		fmt.Println(`{"version":1,"source":"mobilewifi-ios12","status":"unavailable","reason":"denied","devices":[]}`)
	case "pipes":
		child := exec.Command(executable, "--hold-stdout")
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if child.Start() != nil {
			os.Exit(66)
		}
		for range 200 {
			if _, err := os.Stat(executable + ".desc.pid"); err == nil {
				break
			}
			time.Sleep(time.Millisecond)
		}
		fmt.Println(valid)
	default:
		os.Exit(67)
	}
}
