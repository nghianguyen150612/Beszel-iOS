package systems

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"errors"
	"github.com/henrygd/beszel"
	"github.com/henrygd/beszel/internal/common"
	"github.com/henrygd/beszel/internal/entities/monitor"
	"github.com/pocketbase/pocketbase/core"
)

// syncPendingNetworkMonitors runs on WebSocket connect and after successful stats
// fetches. Failed syncs retry on the next update without taking the system down.
func (sys *System) syncPendingNetworkMonitors() {
	if !sys.monitorsNeedSync.Swap(false) {
		return
	}
	if err := sys.syncAllNetworkMonitors(); err != nil {
		sys.monitorsNeedSync.Store(true)
		sys.manager.hub.Logger().Warn("failed to sync monitors to agent", "system", sys.Id, "err", err)
	}
}

func (sys *System) syncAllNetworkMonitors() error {
	configs, err := sys.manager.GetMonitorConfigsForSystem(sys.Id)
	if err != nil {
		return fmt.Errorf("failed to load monitors: %w", err)
	}
	// An empty set must also replace probes retained across a disconnect.
	return sys.SyncNetworkMonitors(configs)
}

// SyncNetworkMonitors sends monitor configurations to the agent.
func (sys *System) SyncNetworkMonitors(configs []monitor.Config) error {
	_, err := sys.syncNetworkMonitors(monitor.SyncRequest{Action: monitor.SyncActionReplace, Configs: configs})
	return err
}

// UpsertNetworkMonitor sends a single monitor configuration change to the agent.
func (sys *System) UpsertNetworkMonitor(config monitor.Config, runNow bool) (*monitor.Result, error) {
	sys.pollMu.Lock()
	defer sys.pollMu.Unlock()
	resp, err := sys.syncNetworkMonitors(monitor.SyncRequest{
		Action: monitor.SyncActionUpsert,
		Config: config,
		RunNow: runNow,
	})
	if err != nil {
		return nil, err
	}
	sys.recordsMu.Lock()
	delete(sys.lastSavedMonitorSample, config.ID)
	sys.recordsMu.Unlock()
	if resp.Result == (monitor.Result{}) {
		return nil, nil
	}
	result := resp.Result
	return &result, nil
}

// DeleteNetworkMonitor removes a single monitor task from the agent.
func (sys *System) DeleteNetworkMonitor(id string) error {
	sys.pollMu.Lock()
	defer sys.pollMu.Unlock()
	_, err := sys.syncNetworkMonitors(monitor.SyncRequest{
		Action: monitor.SyncActionDelete,
		Config: monitor.Config{ID: id},
	})
	return err
}

func (sys *System) syncNetworkMonitors(req monitor.SyncRequest) (monitor.SyncResponse, error) {
	if sys.getAgentVersion().LT(beszel.MinVersionNetworkMonitors) {
		return monitor.SyncResponse{}, nil
	}
	timeout := 5 * time.Second
	if req.Action == monitor.SyncActionUpsert && req.RunNow {
		// Allow the probe to finish, including a timeout result, while preserving
		// the normal request budget for transport and response handling.
		timeout += monitor.MaxProbeTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var result monitor.SyncResponse
	return result, sys.request(ctx, common.SyncNetworkMonitors, req, &result)
}

// freshMonitorResults must be called under recordsMu. SampleCount orders probes
// within a task without assuming a monotonic Agent wall clock. Historical peers
// that omit it retain the existing LastProbeAt identity check.
func (sys *System) freshMonitorResults(results map[string]monitor.Result) map[string]monitor.Result {
	if results == nil {
		return nil
	}
	fresh := make(map[string]monitor.Result, len(results))
	for id, r := range results {
		if r.SampleCount > 0 && sys.lastSavedMonitorSample[id] > 0 && r.SampleCount <= sys.lastSavedMonitorSample[id] {
			continue
		}
		fresh[id] = r
	}
	return fresh
}

func (sys *System) rememberMonitorProbes(probes map[string]int64, results map[string]monitor.Result) {
	if len(probes) == 0 {
		return
	}
	if sys.lastSavedMonitorProbe == nil {
		sys.lastSavedMonitorProbe = make(map[string]int64)
	}
	if sys.lastSavedMonitorSample == nil {
		sys.lastSavedMonitorSample = make(map[string]int64)
	}
	for id, timestamp := range probes {
		sys.lastSavedMonitorProbe[id] = timestamp
		sys.lastSavedMonitorSample[id] = results[id].SampleCount
	}
}

// SaveNetworkMonitorResult publishes an immediate probe only while its exact
// configuration still exists. It shares scheduled history deduplication and
// ordering, so a delayed run-now response cannot recreate a deleted record or
// overwrite a newer result.
func (sys *System) SaveNetworkMonitorResult(config monitor.Config, result monitor.Result) error {
	sys.recordsMu.Lock()
	defer sys.recordsMu.Unlock()
	results := sys.freshMonitorResults(map[string]monitor.Result{config.ID: result})
	saved := make(map[string]int64)
	err := sys.manager.hub.RunInTransaction(func(app core.App) error {
		rec, err := app.FindRecordById("network_monitors", config.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if rec.GetString("system") != sys.Id || !rec.GetBool("enabled") ||
			rec.GetString("target") != config.Target || rec.GetString("protocol") != config.Protocol ||
			rec.GetInt("port") != int(config.Port) || rec.GetInt("interval") != int(config.Interval) ||
			rec.GetString("server") != config.Server || rec.GetBool("skipTlsVerify") != config.SkipTLSVerify {
			return nil
		}
		return sys.updateNetworkMonitorsRecords(app, results, saved)
	})
	if err == nil {
		sys.rememberMonitorProbes(saved, results)
	}
	return err
}
