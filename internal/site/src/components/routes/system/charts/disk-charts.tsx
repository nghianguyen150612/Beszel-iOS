import { t } from "@lingui/core/macro"
import AreaChartDefault from "@/components/charts/area-chart"
import { decimalString, formatBytes, toFixedFloat } from "@/lib/utils"
import { diskDataFns } from "@/lib/disk"
import { ChartCard, SelectAvgMax } from "../chart-card"
import { Unit } from "@/lib/enums"
import DiskIoSheet from "../disk-io-sheet"
import type { SystemData } from "../use-system-data"
import { useStore } from "@nanostores/react"
import { $userSettings } from "@/lib/stores"

export { diskDataFns } from "@/lib/disk"

export function RootDiskCharts({ systemData }: { systemData: SystemData }) {
	return (
		<>
			<DiskUsageChart systemData={systemData} />
			<DiskIOChart systemData={systemData} />
		</>
	)
}

export function DiskUsageChart({ systemData, extraFsName }: { systemData: SystemData; extraFsName?: string }) {
	const { chartData, grid, dataEmpty } = systemData

	let diskSize = chartData.systemStats?.at(-1)?.stats.d ?? NaN
	if (extraFsName) {
		diskSize = chartData.systemStats?.at(-1)?.stats.efs?.[extraFsName]?.d ?? NaN
	}
	// round to nearest GB
	if (diskSize >= 100) {
		diskSize = Math.round(diskSize)
	}

	const rootName = systemData.system?.info?.rdn
	const rootLabel = rootName ?? t({ message: `Root`, context: "Root disk label" })
	const title = extraFsName ? `${extraFsName} ${t`Usage`}` : `${rootLabel} ${t`Usage`}`
	const description = t`Disk usage of ${{ extraFsName: extraFsName ?? rootLabel.toLowerCase() }}`

	return (
		<ChartCard empty={dataEmpty} grid={grid} title={title} description={description}>
			<AreaChartDefault
				chartData={chartData}
				domain={[0, diskSize]}
				tickFormatter={(val) => {
					const { value, unit } = formatBytes(val * 1024, false, Unit.Bytes, true)
					return `${toFixedFloat(value, value >= 10 ? 0 : 1)} ${unit}`
				}}
				contentFormatter={({ value }) => {
					const { value: convertedValue, unit } = formatBytes(value * 1024, false, Unit.Bytes, true)
					return `${decimalString(convertedValue)} ${unit}`
				}}
				dataPoints={[
					{
						label: t`Disk Usage`,
						color: 4,
						opacity: 0.4,
						dataKey: extraFsName ? diskDataFns.extraUsage(extraFsName) : diskDataFns.usage,
					},
				]}
			></AreaChartDefault>
		</ChartCard>
	)
}

export function DiskIOChart({ systemData, extraFsName }: { systemData: SystemData; extraFsName?: string }) {
	const { chartData, grid, dataEmpty, showMax, isLongerChart, maxValues } = systemData
	const maxValSelect = isLongerChart ? <SelectAvgMax max={maxValues} /> : null
	const userSettings = useStore($userSettings)

	if (!chartData.systemStats?.length) {
		return null
	}

	const rootName = systemData.system?.info?.rdn
	const rootLabel = rootName ?? t({ message: `Root`, context: "Root disk label" })
	const title = t`${{ diskName: extraFsName ?? rootLabel }} I/O`
	const description = t`Throughput of ${{ extraFsName: extraFsName ?? rootLabel.toLowerCase() }}`

	const hasMoreIOMetrics = chartData.systemStats?.some((record) => record.stats?.dios?.at(0))

	let CornerEl = maxValSelect
	if (hasMoreIOMetrics) {
		CornerEl = (
			<div className="flex gap-2">
				{maxValSelect}
				<DiskIoSheet systemData={systemData} extraFsName={extraFsName} title={title} description={description} />
			</div>
		)
	}

	let readFn = showMax ? diskDataFns.readMax : diskDataFns.read
	let writeFn = showMax ? diskDataFns.writeMax : diskDataFns.write
	if (extraFsName) {
		readFn = showMax ? diskDataFns.extraReadMax(extraFsName) : diskDataFns.extraRead(extraFsName)
		writeFn = showMax ? diskDataFns.extraWriteMax(extraFsName) : diskDataFns.extraWrite(extraFsName)
	}

	return (
		<ChartCard empty={dataEmpty} grid={grid} title={title} description={description} cornerEl={CornerEl}>
			<AreaChartDefault
				chartData={chartData}
				maxToggled={showMax}
				showTotal={true}
				dataPoints={[
					{
						label: t({ message: "Write", comment: "Disk write" }),
						dataKey: writeFn,
						color: 3,
						opacity: 0.3,
					},
					{
						label: t({ message: "Read", comment: "Disk read" }),
						dataKey: readFn,
						color: 1,
						opacity: 0.3,
					},
				]}
				tickFormatter={(val) => {
					const { value, unit } = formatBytes(val, true, userSettings.unitDisk, false)
					return `${toFixedFloat(value, value >= 10 ? 0 : 1)} ${unit}`
				}}
				contentFormatter={({ value }) => {
					const { value: convertedValue, unit } = formatBytes(value, true, userSettings.unitDisk, false)
					return `${decimalString(convertedValue, convertedValue >= 100 ? 1 : 2)} ${unit}`
				}}
			/>
		</ChartCard>
	)
}

export function DiskUtilizationChart({ systemData, extraFsName }: { systemData: SystemData; extraFsName?: string }) {
	const { chartData, grid, dataEmpty, showMax, isLongerChart, maxValues } = systemData
	const maxValSelect = isLongerChart ? <SelectAvgMax max={maxValues} /> : null

	if (!chartData.systemStats?.length) {
		return null
	}

	let utilFn = showMax ? diskDataFns.utilMax : diskDataFns.util
	if (extraFsName) {
		utilFn = showMax ? diskDataFns.extraUtilMax(extraFsName) : diskDataFns.extraUtil(extraFsName)
	}
	return (
		<ChartCard
			cornerEl={maxValSelect}
			empty={dataEmpty}
			grid={grid}
			title={t({
				message: `I/O Utilization`,
				context: "Percent of time the disk is busy with I/O",
			})}
			description={t`Percent of time the disk is busy with I/O`}
			// legend={true}
			className="min-h-auto"
		>
			<AreaChartDefault
				chartData={chartData}
				tickFormatter={(val) => `${toFixedFloat(val, 2)}%`}
				contentFormatter={({ value }) => `${decimalString(value)}%`}
				maxToggled={showMax}
				chartProps={{ syncId: "io" }}
				dataPoints={[
					{
						label: t({ message: "Utilization", context: "Disk I/O utilization" }),
						dataKey: utilFn,
						color: 1,
						opacity: 0.4,
					},
				]}
			/>
		</ChartCard>
	)
}

export function ExtraFsCharts({ systemData }: { systemData: SystemData }) {
	const { systemStats } = systemData.chartData

	const extraFs = systemStats?.at(-1)?.stats.efs

	if (!extraFs || Object.keys(extraFs).length === 0) {
		return null
	}

	return (
		<div className="grid xl:grid-cols-2 gap-4">
			{Object.keys(extraFs)
				.filter((name) => extraFs[name] != null)
				.sort((a, b) => a.localeCompare(b))
				.map((extraFsName) => {
					let diskSize = systemStats.at(-1)?.stats.efs?.[extraFsName]?.d ?? NaN
					// round to nearest GB
					if (diskSize >= 100) {
						diskSize = Math.round(diskSize)
					}
					return (
						<div key={extraFsName} className="contents">
							<DiskUsageChart systemData={systemData} extraFsName={extraFsName} />

							<DiskIOChart systemData={systemData} extraFsName={extraFsName} />
						</div>
					)
				})}
		</div>
	)
}
