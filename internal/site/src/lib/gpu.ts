import type { GPUData, SystemStatsRecord } from "@/types"

/** Power series use device identity, even when two adapters have the same name. */
export function gpuPowerSeries(records: SystemStatsRecord[]) {
	const series = new Map<string, { id: string; name: string; field: "p" | "pp"; total: number }>()
	for (const record of records) {
		for (const [id, gpu] of Object.entries(record.stats?.g ?? {})) {
			if (!gpu) continue
			for (const field of ["p", "pp"] as const) {
				const value = gpu[field]
				if (typeof value !== "number" || !Number.isFinite(value)) continue
				const key = JSON.stringify([id, gpu.n, field])
				const existing = series.get(key)
				if (existing) existing.total += value
				else series.set(key, { id, name: gpu.n, field, total: value })
			}
		}
	}
	const entries = [...series.values()].sort((a, b) => b.total - a.total || a.id.localeCompare(b.id))
	return entries.map((entry) => {
		const duplicateName = entries.some((other) => other.id !== entry.id && other.name === entry.name)
		return {
			label: `${entry.name}${duplicateName ? ` (${entry.id})` : ""}${entry.field === "pp" ? " package" : ""}`,
			dataKey: (record: SystemStatsRecord): number | null => {
				const gpu: GPUData | undefined = record.stats?.g?.[entry.id]
				const value = gpu?.[entry.field]
				return gpu?.n === entry.name && typeof value === "number" && Number.isFinite(value) ? value : null
			},
		}
	})
}
