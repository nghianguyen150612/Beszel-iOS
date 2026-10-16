//go:build testing

package alerts_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	beszelTests "github.com/henrygd/beszel/internal/tests"
	"github.com/stretchr/testify/require"
)

func TestP014GPUUnavailableDoesNotRecover(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()
	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "paused")
	require.NoError(t, err)
	record := systems[0]
	alert, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{"name": "GPU", "system": record.Id, "user": user.Id, "value": 50, "min": 1})
	require.NoError(t, err)
	for _, sample := range []struct {
		usage              float64
		present, triggered bool
	}{
		{90, true, true}, {90, true, true}, {0, false, true}, {0, true, false},
	} {
		data := &system.CombinedData{Info: system.Info{GpuPct: sample.usage}}
		if sample.present {
			data.Stats.GPUData = map[string]system.GPUData{"gpu0": {Name: "synthetic GPU", Usage: sample.usage}}
		}
		record.Set("updated", time.Now().UTC())
		synctest.Test(t, func(t *testing.T) { require.NoError(t, hub.GetAlertManager().HandleSystemAlerts(record, data)) })
		saved, err := hub.FindRecordById("alerts", alert.Id)
		require.NoError(t, err)
		require.Equal(t, sample.triggered, saved.GetBool("triggered"), "present=%v usage=%v", sample.present, sample.usage)
	}
}

func TestP014TemperatureZeroRecovery(t *testing.T) {
	hub, user := beszelTests.GetHubWithUser(t)
	defer hub.Cleanup()
	systems, err := beszelTests.CreateSystems(hub, 1, user.Id, "paused")
	require.NoError(t, err)
	record := systems[0]
	alert, err := beszelTests.CreateRecord(hub, "alerts", map[string]any{"name": "Temperature", "system": record.Id, "user": user.Id, "value": 50, "min": 1})
	require.NoError(t, err)
	for _, sample := range []struct {
		temp               float64
		present, triggered bool
	}{{90, true, true}, {0, false, true}, {0, true, false}} {
		data := &system.CombinedData{Info: system.Info{DashboardTemp: sample.temp}}
		if sample.present {
			data.Stats.Temperatures = map[string]float64{"sensor0": sample.temp}
		}
		record.Set("updated", time.Now().UTC())
		synctest.Test(t, func(t *testing.T) { require.NoError(t, hub.GetAlertManager().HandleSystemAlerts(record, data)) })
		saved, err := hub.FindRecordById("alerts", alert.Id)
		require.NoError(t, err)
		require.Equal(t, sample.triggered, saved.GetBool("triggered"))
	}
}
