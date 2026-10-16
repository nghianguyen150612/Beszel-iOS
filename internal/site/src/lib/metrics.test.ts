import { expect, test } from "bun:test"
import type { SystemStatsRecord } from "@/types"
import { diskDataFns } from "./disk"
import { byteRate, coreUsage, metricValue, poolByteRate } from "./metrics"

const record = (stats: unknown) => ({ stats }) as SystemStatsRecord

test("disk charts keep absent/null/empty optional data as gaps", () => {
	for (const stats of [undefined, null, {}, { efs: null }, { efs: { missing: null } }]) {
		const row = record(stats)
		for (const fn of [diskDataFns.read, diskDataFns.write, diskDataFns.readTime, diskDataFns.weightedIO, diskDataFns.totalRead, diskDataFns.extraRead("missing"), diskDataFns.extraUsage("missing")]) {
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

test("missing cores and pools never gain fallback measurements", () => {
	expect(coreUsage(record({ cpus: [0] }), 0)).toBe(0)
	expect(coreUsage(record({ cpus: [] }), 0)).toBeNull()
	expect(coreUsage(record({}), 0)).toBeNull()
	expect(poolByteRate(record({ z: { pool: { d: 100, du: 0 } } }), "pool", "rb")).toBe(0)
	expect(poolByteRate(record({ z: {} }), "pool", "rb")).toBeNull()
	expect(poolByteRate(record({ z: { pool: null } }), "pool", "rb")).toBeNull()
	expect(poolByteRate(record({ z: { pool: {} } }), "pool", "rb")).toBeNull()
})
