import type { SystemStatsRecord } from "@/types"
import { byteRate, metricValue, scaleMetric } from "./metrics"

// Helpers for indexed dios/diosm access
const dios =
	(i: number) =>
	({ stats }: SystemStatsRecord) =>
		metricValue(stats?.dios?.[i])
const diosMax =
	(i: number) =>
	({ stats }: SystemStatsRecord) =>
		metricValue(stats?.diosm?.[i])
const extraDios =
	(name: string, i: number) =>
	({ stats }: SystemStatsRecord) =>
		metricValue(stats?.efs?.[name]?.dios?.[i])
const extraDiosMax =
	(name: string, i: number) =>
	({ stats }: SystemStatsRecord) =>
		metricValue(stats?.efs?.[name]?.diosm?.[i])

export const diskDataFns = {
	// usage
	usage: ({ stats }: SystemStatsRecord) => metricValue(stats?.du),
	extraUsage:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			metricValue(stats?.efs?.[name]?.du),
	// throughput
	read: ({ stats }: SystemStatsRecord) => byteRate(stats?.dio?.[0], stats?.dr),
	readMax: ({ stats }: SystemStatsRecord) => byteRate(stats?.diom?.[0], stats?.drm),
	write: ({ stats }: SystemStatsRecord) => byteRate(stats?.dio?.[1], stats?.dw),
	writeMax: ({ stats }: SystemStatsRecord) => byteRate(stats?.diom?.[1], stats?.dwm),
	// extra fs throughput
	extraRead:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			byteRate(stats?.efs?.[name]?.rb, stats?.efs?.[name]?.r),
	extraReadMax:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			byteRate(stats?.efs?.[name]?.rbm, stats?.efs?.[name]?.rm),
	extraWrite:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			byteRate(stats?.efs?.[name]?.wb, stats?.efs?.[name]?.w),
	extraWriteMax:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			byteRate(stats?.efs?.[name]?.wbm, stats?.efs?.[name]?.wm),
	// cumulative totals
	totalRead: ({ stats }: SystemStatsRecord) => metricValue(stats?.diot?.[0]),
	totalWrite: ({ stats }: SystemStatsRecord) => metricValue(stats?.diot?.[1]),
	extraTotalRead:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			metricValue(stats?.efs?.[name]?.tr),
	extraTotalWrite:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			metricValue(stats?.efs?.[name]?.tw),
	// read/write time
	readTime: dios(0),
	readTimeMax: diosMax(0),
	extraReadTime: (name: string) => extraDios(name, 0),
	extraReadTimeMax: (name: string) => extraDiosMax(name, 0),
	writeTime: dios(1),
	writeTimeMax: diosMax(1),
	extraWriteTime: (name: string) => extraDios(name, 1),
	extraWriteTimeMax: (name: string) => extraDiosMax(name, 1),
	// utilization (IoTime-based, 0-100%)
	util: dios(2),
	utilMax: diosMax(2),
	extraUtil: (name: string) => extraDios(name, 2),
	extraUtilMax: (name: string) => extraDiosMax(name, 2),
	// r_await / w_await: average service time per read/write operation (ms)
	rAwait: dios(3),
	rAwaitMax: diosMax(3),
	extraRAwait: (name: string) => extraDios(name, 3),
	extraRAwaitMax: (name: string) => extraDiosMax(name, 3),
	wAwait: dios(4),
	wAwaitMax: diosMax(4),
	extraWAwait: (name: string) => extraDios(name, 4),
	extraWAwaitMax: (name: string) => extraDiosMax(name, 4),
	// average queue depth: stored as queue_depth * 100 in Go, divided here
	weightedIO: ({ stats }: SystemStatsRecord) => scaleMetric(stats?.dios?.[5], 0.01),
	weightedIOMax: ({ stats }: SystemStatsRecord) => scaleMetric(stats?.diosm?.[5], 0.01),
	extraWeightedIO:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			scaleMetric(stats?.efs?.[name]?.dios?.[5], 0.01),
	extraWeightedIOMax:
		(name: string) =>
		({ stats }: SystemStatsRecord) =>
			scaleMetric(stats?.efs?.[name]?.diosm?.[5], 0.01),
}
