import { expect, test } from "bun:test"
import type { SystemStatsRecord } from "@/types"
import { gpuPowerSeries } from "./gpu"

const row = (g: unknown) => ({ stats: { g } }) as SystemStatsRecord

test("GPU power preserves zero and gaps without mixing equal-name devices", () => {
	const records = [row({ a: { n: "Adapter", p: 0, pp: 7.5 }, b: { n: "Adapter", p: 20 } }), row(null), row({})]
	const series = gpuPowerSeries(records)
	expect(series).toHaveLength(3)
	const a = series.find((s) => s.label === "Adapter (a)")!
	const b = series.find((s) => s.label === "Adapter (b)")!
	expect(a.dataKey(records[0])).toBe(0)
	expect(b.dataKey(records[0])).toBe(20)
	for (const s of series) {
		expect(s.dataKey(records[1])).toBeNull()
		expect(s.dataKey(records[2])).toBeNull()
		expect(s.dataKey(row({ a: { n: "replacement", p: 5 } }))).toBeNull()
	}
})

test("GPU power ignores unavailable and invalid readings", () => {
	expect(gpuPowerSeries([row(undefined), row({ a: null }), row({ a: { n: "A", p: Number.NaN } })])).toEqual([])
})
