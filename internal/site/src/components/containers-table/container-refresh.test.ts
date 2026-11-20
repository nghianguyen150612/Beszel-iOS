import { expect, test } from "bun:test"
import {
	createTable,
	getCoreRowModel,
	getFilteredRowModel,
	getPaginationRowModel,
	getSortedRowModel,
} from "@tanstack/react-table"
import {
	ContainerRefreshCoordinator,
	mergeSystemContainers,
	selectRecentContainers,
	type ContainerRowSnapshot,
} from "./container-refresh"

type Row = ContainerRowSnapshot & { name: string }

function row(id: string, system: string, updated: number, name = id): Row {
	return { id, system, updated, name }
}

function deferred<T>() {
	let resolve!: (value: T) => void
	let reject!: (error: unknown) => void
	const promise = new Promise<T>((res, rej) => {
		resolve = res
		reject = rej
	})
	return { promise, resolve, reject }
}

async function flushMicrotasks() {
	await Promise.resolve()
	await Promise.resolve()
	await Promise.resolve()
}

function makeCoordinator(fetch: (systemId: string | undefined, signal: AbortSignal) => Promise<Row[]>) {
	const initial: Row[][] = []
	const systems: Array<[string, Row[]]> = []
	const removed: string[] = []
	const errors: unknown[] = []
	const coordinator = new ContainerRefreshCoordinator<Row>({
		fetch,
		onInitial: (items) => initial.push(items),
		onSystem: (systemId, items) => systems.push([systemId, items]),
		onSystemRemoved: (systemId) => removed.push(systemId),
		onError: (error) => errors.push(error),
	})
	return { coordinator, initial, systems, removed, errors }
}

test("different systems can refresh concurrently without cancelling each other", async () => {
	const requests = new Map<string, { signal: AbortSignal; response: ReturnType<typeof deferred<Row[]>> }>()
	const { coordinator, systems } = makeCoordinator((systemId, signal) => {
		const response = deferred<Row[]>()
		requests.set(systemId!, { signal, response })
		return response.promise
	})
	coordinator.startSystem("alpha")
	coordinator.refreshSystem("beta")

	expect(requests.get("alpha")?.signal.aborted).toBe(false)
	expect(requests.get("beta")?.signal.aborted).toBe(false)
	requests.get("beta")?.response.resolve([row("b1", "beta", 100)])
	requests.get("alpha")?.response.resolve([row("a1", "alpha", 100)])
	await flushMicrotasks()
	expect(systems.map(([system]) => system).sort()).toEqual(["alpha", "beta"])
	coordinator.dispose()
})

test("cancelling one system request leaves another system request active", async () => {
	const requests: Array<{ systemId: string | undefined; signal: AbortSignal; response: ReturnType<typeof deferred<Row[]>> }> = []
	const { coordinator, systems } = makeCoordinator((systemId, signal) => {
		const response = deferred<Row[]>()
		requests.push({ systemId, signal, response })
		return response.promise
	})
	coordinator.startSystem("alpha")
	coordinator.refreshSystem("alpha")
	coordinator.refreshSystem("beta")

	expect(requests[0].signal.aborted).toBe(true)
	expect(requests[1].signal.aborted).toBe(false)
	expect(requests[2].signal.aborted).toBe(false)
	requests[0].response.resolve([row("stale", "alpha", 100)])
	requests[1].response.resolve([row("new", "alpha", 200)])
	requests[2].response.resolve([row("other", "beta", 200)])
	await flushMicrotasks()
	expect(systems.map(([, items]) => items[0].id).sort()).toEqual(["new", "other"])
	coordinator.dispose()
})

test("out-of-order same-system responses cannot overwrite the newest request", async () => {
	const requests: Array<ReturnType<typeof deferred<Row[]>>> = []
	const { coordinator, systems } = makeCoordinator(() => {
		const response = deferred<Row[]>()
		requests.push(response)
		return response.promise
	})
	coordinator.startSystem("alpha")
	coordinator.refreshSystem("alpha")
	requests[1].resolve([row("new", "alpha", 200)])
	requests[0].resolve([row("old", "alpha", 100)])
	await flushMicrotasks()
	expect(systems).toEqual([["alpha", [row("new", "alpha", 200)]]])
	coordinator.dispose()
})

test("All Containers waits for its first snapshot before applying queued system refreshes", async () => {
	const initial = deferred<Row[]>()
	const refresh = deferred<Row[]>()
	const calls: Array<string | undefined> = []
	const { coordinator, initial: snapshots, systems } = makeCoordinator((systemId) => {
		calls.push(systemId)
		return systemId ? refresh.promise : initial.promise
	})
	coordinator.startAll()
	coordinator.refreshSystem("alpha")
	coordinator.refreshSystem("alpha")
	coordinator.refreshSystem("beta")
	expect(calls).toEqual([undefined])

	initial.resolve([row("old", "alpha", 100)])
	await flushMicrotasks()
	expect(snapshots).toEqual([[row("old", "alpha", 100)]])
	expect(calls).toEqual([undefined, "alpha", "beta"])
	refresh.resolve([row("new", "alpha", 200)])
	await flushMicrotasks()
	expect(systems).toContainEqual(["alpha", [row("new", "alpha", 200)]])
	coordinator.dispose()
})

test("a system removed during the initial All Containers request stays removed", async () => {
	const initial = deferred<Row[]>()
	const { coordinator, initial: snapshots, removed } = makeCoordinator(() => initial.promise)
	coordinator.startAll()
	coordinator.removeSystem("deleted")
	initial.resolve([row("gone", "deleted", 100), row("kept", "active", 100)])
	await flushMicrotasks()
	expect(snapshots).toEqual([[row("kept", "active", 100)]])
	expect(removed).toEqual(["deleted"])
	coordinator.dispose()
})

test("a system re-added while the initial request is pending is refreshed after it", async () => {
	const initial = deferred<Row[]>()
	const refresh = deferred<Row[]>()
	const calls: Array<string | undefined> = []
	const { coordinator, initial: snapshots, systems } = makeCoordinator((systemId) => {
		calls.push(systemId)
		return systemId ? refresh.promise : initial.promise
	})
	coordinator.startAll()
	coordinator.removeSystem("alpha")
	coordinator.refreshSystem("alpha")
	initial.resolve([row("initial", "alpha", 100)])
	await flushMicrotasks()
	expect(snapshots).toEqual([[row("initial", "alpha", 100)]])
	expect(calls).toEqual([undefined, "alpha"])
	refresh.resolve([row("current", "alpha", 200)])
	await flushMicrotasks()
	expect(systems).toEqual([["alpha", [row("current", "alpha", 200)]]])
	coordinator.dispose()
})

test("rapid system switching disposes old work and accepts only the selected system", async () => {
	const alpha = deferred<Row[]>()
	const beta = deferred<Row[]>()
	let alphaSignal: AbortSignal | undefined
	const oldTable = makeCoordinator((_systemId, signal) => {
		alphaSignal = signal
		return alpha.promise
	})
	oldTable.coordinator.startSystem("alpha")
	oldTable.coordinator.dispose()
	const newTable = makeCoordinator((_systemId, _signal) => beta.promise)
	newTable.coordinator.startSystem("beta")

	expect(alphaSignal?.aborted).toBe(true)
	alpha.resolve([row("a", "alpha", 100)])
	beta.resolve([row("b", "beta", 100)])
	await flushMicrotasks()
	expect(oldTable.systems).toEqual([])
	expect(newTable.systems).toEqual([["beta", [row("b", "beta", 100)]]])
	newTable.coordinator.dispose()
})

test("empty and partially loaded snapshots update only the requested system", () => {
	const current = [row("old-a", "alpha", 80_000), row("keep-b", "beta", 99_000)]
	const partial = mergeSystemContainers(current, "alpha", [row("new-a", "alpha", 100_000)])
	expect(partial.map(({ id }) => id)).toEqual(["new-a", "old-a", "keep-b"])
	expect(mergeSystemContainers(partial, "alpha", []).map(({ id }) => id)).toEqual(["keep-b"])
	expect(selectRecentContainers<Row>([])).toEqual([])
})

test("refresh errors preserve data and a later system update retries", async () => {
	let attempt = 0
	const { coordinator, systems, errors } = makeCoordinator(async () => {
		attempt += 1
		if (attempt === 1) throw new Error("temporary network failure")
		return [row("retry-ok", "alpha", 100)]
	})
	const visible = [row("visible", "alpha", 50)]
	coordinator.startSystem("alpha")
	await flushMicrotasks()
	expect(errors).toHaveLength(1)
	expect(systems).toEqual([])
	expect(visible).toEqual([row("visible", "alpha", 50)])
	coordinator.refreshSystem("alpha")
	await flushMicrotasks()
	expect(systems).toEqual([["alpha", [row("retry-ok", "alpha", 100)]]])
	expect(errors).toHaveLength(1)
	coordinator.dispose()
})

test("refreshed rows retain table sorting and filtering", () => {
	const table = createTable({
		data: [row("a", "alpha", 10, "zeta"), row("b", "beta", 10, "beta")],
		columns: [{ accessorKey: "name" }, { accessorKey: "system" }],
		state: {
			sorting: [{ id: "name", desc: false }],
			columnFilters: [{ id: "system", value: "alpha" }],
			pagination: { pageIndex: 0, pageSize: 10 },
			globalFilter: "",
		},
		onStateChange: () => {},
		renderFallbackValue: null,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		autoResetPageIndex: false,
	})
	table.setOptions((options) => ({ ...options, data: [row("c", "alpha", 20, "alpha"), row("a", "alpha", 10, "zeta")] }))
	expect(table.getRowModel().rows.map(({ original }) => original.name)).toEqual(["alpha", "zeta"])
})

test("refreshing data preserves the shared pagination index", () => {
	const table = createTable({
		data: [row("a", "alpha", 10), row("b", "alpha", 10), row("c", "alpha", 10), row("d", "alpha", 10)],
		columns: [{ accessorKey: "id" }],
		state: { sorting: [], columnFilters: [], pagination: { pageIndex: 1, pageSize: 2 }, globalFilter: "" },
		onStateChange: () => {},
		renderFallbackValue: null,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		autoResetPageIndex: false,
	})
	table.setOptions((options) => ({ ...options, data: [row("a", "alpha", 20), row("b", "alpha", 20), row("c", "alpha", 20), row("d", "alpha", 20)] }))
	expect(table.getState().pagination.pageIndex).toBe(1)
	expect(table.getRowModel().rows.map(({ original }) => original.id)).toEqual(["c", "d"])
})

test("unmount aborts each owned request and ignores their later responses", async () => {
	const requests: Array<{ signal: AbortSignal; response: ReturnType<typeof deferred<Row[]>> }> = []
	const { coordinator, systems } = makeCoordinator((_systemId, signal) => {
		const response = deferred<Row[]>()
		requests.push({ signal, response })
		return response.promise
	})
	coordinator.startSystem("alpha")
	coordinator.refreshSystem("beta")
	coordinator.dispose()
	expect(requests.every(({ signal }) => signal.aborted)).toBe(true)
	for (const request of requests) request.response.resolve([row("late", "alpha", 100)])
	await flushMicrotasks()
	expect(systems).toEqual([])
})
