import { expect, test } from "bun:test"
import type { SystemStatsRecord } from "@/types"
import { diskDataFns } from "./disk"
import { byteRate, coreUsage, metricValue, poolByteRate } from "./metrics"

const record = (stats: unknown) => ({ stats }) as SystemStatsRecord

test("disk charts keep absent/null/empty optional data as gaps", () => {
	for (const stats of [undefined, null, {}, { efs: null }, { efs: { missing: null } }]) {
		const row = record(stats)
		for (const fn of [
			diskDataFns.read,
			diskDataFns.write,
			diskDataFns.readTime,
			diskDataFns.weightedIO,
			diskDataFns.totalRead,
			diskDataFns.extraRead("missing"),
			diskDataFns.extraUsage("missing"),
		]) {
			expect(fn(row)).toBeNull()
		}
	}
})

test("disk and network codecs preserve measured zero and legacy units", () => {
	const row = record({ dio: [0, 2048], dios: [0, 0, 0, 0, 0, 50], efs: { disk: { d: 100, du: 0, r: 0, w: 1 } } })
	expect(diskDataFns.read(row)).toBe(0)
	expect(diskDataFns.write(row)).toBe(2048)
	expect(diskDataFns.weightedIO(row)).toBe(0.5)
	expect(diskDataFns.extraUsage("disk")(row)).toBe(0)
	expect(diskDataFns.extraRead("disk")(row)).toBe(0)
	expect(diskDataFns.extraWrite("disk")(row)).toBe(1024 * 1024)
	expect(byteRate(undefined, 1.5)).toBe(1.5 * 1024 * 1024)
	expect(byteRate(0, 5)).toBe(0)
	expect(byteRate(undefined, undefined)).toBeNull()
	for (const value of [undefined, null, Number.NaN, Infinity, "0"]) expect(metricValue(value)).toBeNull()
})

test("disk throughput falls back to legacy byte rates for older Agent telemetry", () => {
	const legacy = record({
		dr: 2,
		drm: 4,
		dw: 8,
		dwm: 16,
		efs: { old: { d: 100, du: 1, r: 2, rm: 4, w: 8, wm: 16 } },
	})
	expect(diskDataFns.read(legacy)).toBe(2 * 1024 * 1024)
	expect(diskDataFns.readMax(legacy)).toBe(4 * 1024 * 1024)
	expect(diskDataFns.write(legacy)).toBe(8 * 1024 * 1024)
	expect(diskDataFns.writeMax(legacy)).toBe(16 * 1024 * 1024)
	expect(diskDataFns.extraRead("old")(legacy)).toBe(2 * 1024 * 1024)
	expect(diskDataFns.extraReadMax("old")(legacy)).toBe(4 * 1024 * 1024)
	expect(diskDataFns.extraWrite("old")(legacy)).toBe(8 * 1024 * 1024)
	expect(diskDataFns.extraWriteMax("old")(legacy)).toBe(16 * 1024 * 1024)

	const current = record({ dio: [512, 1024], dr: 2, dw: 4 })
	expect(diskDataFns.read(current)).toBe(512)
	expect(diskDataFns.write(current)).toBe(1024)
})

test("missing cores and pools never gain fallback measurements", () => {
	expect(coreUsage(record({ cpus: [0] }), 0)).toBe(0)
	expect(coreUsage(record({ cpus: [] }), 0)).toBeNull()
	expect(coreUsage(record({}), 0)).toBeNull()
	expect(poolByteRate(record({ z: { pool: { d: 100, du: 0 } } }), "pool", "rb")).toBe(0)
	expect(poolByteRate(record({ z: {} }), "pool", "rb")).toBeNull()
	expect(poolByteRate(record({ z: { pool: null } }), "pool", "rb")).toBeNull()
	expect(poolByteRate(record({ z: { pool: {} } }), "pool", "rb")).toBeNull()
})
