//go:build testing

package agent

import (
	"errors"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/stretchr/testify/require"
)

func TestP010IOSUnsupportedCollectors(t *testing.T) {
	// Even PATH tools or forced GPU selection cannot activate unrelated iOS tools.
	t.Setenv("GPU_COLLECTOR", "nvidia-smi,powermetrics,macmon")
	t.Setenv("SMART_DEVICES", "/dev/fake")
	for range 10 {
		gm, err := newGPUManagerForPlatform("ios")
		require.Nil(t, gm)
		require.ErrorIs(t, err, errors.ErrUnsupported)
		sm, err := newSmartManagerForPlatform("ios")
		require.Nil(t, sm)
		require.ErrorIs(t, err, errors.ErrUnsupported)
	}
	pool := newStoragePoolManagerForPlatform("ios")
	require.Empty(t, pool.backends)
	stats := &system.Stats{ZfsPools: map[string]*system.ZfsPool{"stale": {Total: 123}}}
	pool.Update(stats)
	require.Nil(t, stats.ZfsPools)
	require.Nil(t, pool.DatasetUsage())
	detail := pool.GetDetail(true)
	require.Empty(t, detail.Pools)
	require.True(t, detail.Complete)
	wire, err := cbor.Marshal(stats)
	require.NoError(t, err)
	var decoded system.Stats
	require.NoError(t, cbor.Unmarshal(wire, &decoded))
	require.Empty(t, decoded.ZfsPools)
	require.Empty(t, decoded.GPUData)
	require.Empty(t, decoded.Temperatures)
	require.Empty(t, decoded.Fans)
}

func TestP010IOSDockerExplicitRemoteOnly(t *testing.T) {
	// Use GetEnv's actual presence rule; the test runs independently of the host OS.
	t.Setenv("DOCKER_HOST", "")
	require.Nil(t, newDockerManagerForPlatform(&Agent{}, "ios"))
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	manager := newDockerManagerForPlatform(&Agent{}, "ios")
	require.NotNil(t, manager)
	require.Positive(t, manager.client.Timeout)
}
