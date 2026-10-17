import type { SystemStatsRecord } from "@/types"

/** Missing/invalid readings are chart gaps; an explicit finite zero is a value. */
export function metricValue(value: unknown): number | null {
	return typeof value === "number" && Number.isFinite(value) ? value : null
}

export function scaleMetric(value: unknown, factor: number): number | null {
	const reading = metricValue(value)
	return reading === null ? null : reading * factor
}

/** Preserve old MiB/s records without inventing a value when both fields are absent. */
export function byteRate(bytes: unknown, legacyMiB: unknown): number | null {
	return metricValue(bytes) ?? scaleMetric(legacyMiB, 1024 * 1024)
}

export function coreUsage(record: SystemStatsRecord, index: number): number | null {
	return metricValue(record.stats?.cpus?.[index])
}

/** A valid pool snapshot reports both directions; omitted zero counters retain that schema meaning. */
export function poolByteRate(record: SystemStatsRecord, name: string, direction: "rb" | "wb"): number | null {
	const pool = record.stats?.z?.[name]
	if (!pool || metricValue(pool.d) === null || metricValue(pool.du) === null) return null
	return metricValue(pool[direction] ?? 0)
}
