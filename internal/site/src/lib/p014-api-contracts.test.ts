import { expect, test } from "bun:test"
import { readFileSync } from "node:fs"
import type { SystemInfo, SystemStatsRecord, SystemRecord } from "@/types"
import { connectedWiFi, strongestWiFiSignal } from "./wifi"
import { gpuPowerSeries } from "./gpu"
import { clearFailedResponse, getMonitorStats, mergeMonitorStats, withMonitorGaps } from "./network-monitor-utils"

// Golden files come from authenticated requests to the real Hub test router,
// after signature verification, scheduled polling, CBOR decoding and DB writes.
for (const platform of ["linux", "windows", "macos", "ios"]) {
	test(`real Hub API snapshot: ${platform}`, () => {
		const snapshot = JSON.parse(
			readFileSync(new URL(`../../../hub/testdata/p014/${platform}-api.json`, import.meta.url), "utf8")
		)
		const info: SystemInfo = snapshot.info
		const row = { stats: snapshot.stats } as SystemStatsRecord
		expect(info.cpu).toBe(0)
		expect(row.stats.cpu).toBe(0)
		expect(row.stats.cpus).toEqual([0, 25, 50, 100])
		const system = { status: "up", info } as SystemRecord
		if (platform === "ios") {
			expect(connectedWiFi(system)).toEqual([])
			expect(gpuPowerSeries([row])).toEqual([])
			expect(row.stats.bat).toBeUndefined()
			expect(row.stats.t).toBeUndefined()
			expect(info.pu).toBeUndefined()
		} else {
			expect(strongestWiFiSignal(system)).toBe(-60)
			expect(row.stats.bat).toEqual([0, 1])
			expect(row.stats.t?.sensor0).toBe(0)
			const power = gpuPowerSeries([row])
			expect(power.map((s) => s.dataKey(row)).sort((a, b) => a! - b!)).toEqual([3.25, 7.5])
		}
		if (platform === "linux") {
			expect(info.pu).toEqual([0, 0])
			expect(snapshot.packages.packages).toEqual([])
			expect(snapshot.packages.securityKnown).toBe(true)
		}
		if (platform === "linux" || platform === "windows") {
			expect(snapshot.containers).toHaveLength(1)
			expect(snapshot.containers[0].c).toBe(0)
			expect(snapshot.containers[0].m).toBe(12.5)
		} else expect(snapshot.containers).toBeUndefined()
	})
}

test("missing monitor history and failed windows never create latency", () => {
	expect(mergeMonitorStats([])).toEqual([])
	expect(withMonitorGaps([], { id: "deleted", interval: 60 }, 60_000)).toEqual([])
	const failed = getMonitorStats({ total_count: 3, success_count: 0, res_sum: 0, res_min: 0, res_max: 0 } as Parameters<
		typeof getMonitorStats
	>[0])
	expect(failed).toEqual({ res_avg: null, res_min: null, res_max: null, loss: 100 })
	expect(clearFailedResponse({ res_avg: 0, res_min: 0, res_max: 0, loss: 100 })).toEqual(failed)
	const zero = getMonitorStats({ total_count: 1, success_count: 1, res_sum: 0, res_min: 0, res_max: 0 } as Parameters<
		typeof getMonitorStats
	>[0])
	expect(zero.res_avg).toBe(0)
})
