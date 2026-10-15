//go:build testing && ios

package zfs

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIOSZfsNeverExecutesUtilities(t *testing.T) {
	previous := commandOutput
	commandOutput = func(string, ...string) ([]byte, error) {
		t.Fatal("unsupported iOS collector invoked a utility")
		return nil, nil
	}
	t.Cleanup(func() { commandOutput = previous })
	_, err := PoolStats()
	require.ErrorIs(t, err, errors.ErrUnsupported)
	_, err = Datasets()
	require.ErrorIs(t, err, errors.ErrUnsupported)
	_, err = PoolStatuses()
	require.ErrorIs(t, err, errors.ErrUnsupported)
}
