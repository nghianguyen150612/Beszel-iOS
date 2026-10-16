package ioshelper

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
)

var (
	ErrBusy        = errors.New("Wi-Fi helper acquisition already in progress")
	ErrClosed      = errors.New("Wi-Fi helper client closed")
	ErrOutputLimit = errors.New("Wi-Fi helper output limit exceeded")
	ErrExecutable  = errors.New("Wi-Fi helper must be a regular executable at an absolute path")
)

// Client is an inactive, bounded subprocess boundary for a future trusted helper.
// It never discovers binaries via PATH or uses a shell. P016 must establish the
// bundled helper's identity/signature; NewClient is not an authorization gate.
// Normal Agent polling does not instantiate or import this client.
type Client struct {
	path      string
	mu        sync.Mutex
	closed    bool
	cancel    context.CancelFunc
	done      chan struct{}
	timeout   time.Duration
	waitDelay time.Duration
	exists    func(string) bool
}

// NewClient accepts a missing absolute helper path; availability is checked on
// each acquisition, so absence never makes the Agent fail to start.
func NewClient(path string) (*Client, error) {
	if !filepath.IsAbs(path) {
		return nil, ErrExecutable
	}
	return &Client{path: path, timeout: 2 * time.Second, waitDelay: 100 * time.Millisecond, exists: interfaceExists}, nil
}

// Read permits one active helper per client and rejects concurrent callers
// immediately. Every path returns a fresh snapshot or nil, never cached data.
func (c *Client) Read(parent context.Context) (map[string]system.WiFi, error) {
	if err := parent.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, ErrClosed
	}
	if c.done != nil {
		c.mu.Unlock()
		return nil, ErrBusy
	}
	ctx, cancel := context.WithTimeout(parent, c.timeout)
	done := make(chan struct{})
	c.cancel = cancel
	c.done = done
	c.mu.Unlock()
	defer func() {
		cancel()
		c.mu.Lock()
		c.cancel = nil
		c.done = nil
		close(done)
		c.mu.Unlock()
	}()
	info, err := os.Lstat(c.path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return nil, ErrExecutable
	}
	cmd := exec.CommandContext(ctx, c.path, "--snapshot-v1")
	cmd.Env = []string{"LC_ALL=C", "LANG=C"}
	// nil stdin/stderr use the null device: no shell, inherited secrets or
	// unbounded stderr pump. Only bounded stdout is copied by os/exec.
	cmd.WaitDelay = c.waitDelay
	if err := configureProcess(cmd); err != nil {
		return nil, err
	}
	output := &boundedOutput{cancel: cancel}
	cmd.Stdout = output
	err = cmd.Run()
	// Also remove descendants retaining stdout after the direct child exits.
	// The trusted native helper is single-process and must never spawn children.
	cleanupErr := cleanupProcess(cmd)
	if output.exceeded {
		return nil, ErrOutputLimit
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	if cleanupErr != nil {
		return nil, cleanupErr
	}
	return parseSnapshot(output.data, c.exists)
}

// Close prevents new acquisitions, cancels the active process group, and waits
// for the acquisition's os/exec Wait/reaping path. The caller can bound shutdown
// waiting explicitly; a timeout is reported, never described as successful reap.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	c.closed = true
	if c.cancel != nil {
		c.cancel()
	}
	done := c.done
	c.mu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type boundedOutput struct {
	data     []byte
	exceeded bool
	cancel   context.CancelFunc
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	remaining := MaxOutputBytes - len(b.data)
	if len(p) > remaining {
		b.data = append(b.data, p[:remaining]...)
		b.exceeded = true
		b.cancel()
		return remaining, ErrOutputLimit
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
