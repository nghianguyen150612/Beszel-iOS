import { expect, test } from "bun:test"
import { createTable, getCoreRowModel, getFilteredRowModel, getPaginationRowModel, getSortedRowModel } from "@tanstack/react-table"
import { getPageBounds, normalizePageSize, pageSizes, paginationLayoutClassName, resetVirtualizedScrollPosition, savePageSize } from "./pagination"

test("pagination reports empty, first, partial, and clamped final page boundaries", () => {
	expect(getPageBounds(0, 0, 20)).toEqual({ pageIndex: 0, pageCount: 0, start: 0, end: 0 })
	expect(getPageBounds(20, 0, 20)).toEqual({ pageIndex: 0, pageCount: 1, start: 0, end: 20 })
	expect(getPageBounds(21, 0, 20)).toEqual({ pageIndex: 0, pageCount: 2, start: 0, end: 20 })
	expect(getPageBounds(21, 1, 20)).toEqual({ pageIndex: 1, pageCount: 2, start: 20, end: 21 })
	expect(getPageBounds(21, 99, 20)).toEqual({ pageIndex: 1, pageCount: 2, start: 20, end: 21 })
})

test("page size accepts supported values and persists a safe default for invalid values", () => {
	for (const size of pageSizes) expect(normalizePageSize(size)).toBe(size)
	expect(normalizePageSize("50")).toBe(50)
	expect(normalizePageSize("not a number")).toBe(20)
	expect(normalizePageSize(7)).toBe(20)
	const saved = new Map<string, string>()
	expect(savePageSize({ setItem: (key, value) => saved.set(key, value) }, 7, () => {})).toBe(20)
	expect(saved.get("besz-rows-per-page")).toBe("20")
})

test("page size updates the local preference and syncs authenticated user settings", () => {
	const saved = new Map<string, string>()
	const updates: number[] = []
	const queued: number[] = []
	const pageSize = savePageSize(
		{ setItem: (key, value) => saved.set(key, value) },
		100,
		(value) => updates.push(value),
		(value) => queued.push(value)
	)
	expect(pageSize).toBe(100)
	expect(saved.get("besz-rows-per-page")).toBe("100")
	expect(updates).toEqual([100])
	expect(queued).toEqual([100])
})

test("page size keeps the in-memory preference when browser storage throws", () => {
	const updates: number[] = []
	const pageSize = savePageSize(
		{ setItem: () => { throw new Error("storage disabled") } },
		50,
		(value) => updates.push(value)
	)
	expect(pageSize).toBe(50)
	expect(updates).toEqual([50])
})

test("sorting and filtering are applied before page boundaries on a large table", () => {
	const systems = Array.from({ length: 10_000 }, (_, index) => ({ id: index, name: `system-${index}` }))
	const table = createTable({
		data: systems,
		columns: [{ accessorKey: "name" }, { accessorKey: "id" }],
		state: {
			pagination: { pageIndex: 1, pageSize: 50 },
			sorting: [{ id: "id", desc: true }],
			globalFilter: "",
			columnFilters: [],
		},
		onStateChange: () => {},
		renderFallbackValue: null,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
	})
	const page = table.getRowModel().rows
	expect(page).toHaveLength(50)
	expect(page[0].original.id).toBe(9_949)
	expect(page[49].original.id).toBe(9_900)
	expect(table.getPrePaginationRowModel().rows).toHaveLength(10_000)

	const filteredTable = createTable({
		data: systems,
		columns: [{ accessorKey: "name" }, { accessorKey: "id" }],
		state: {
			pagination: { pageIndex: 0, pageSize: 50 },
			sorting: [{ id: "id", desc: true }],
			globalFilter: "system-9",
			columnFilters: [],
		},
		onStateChange: () => {},
		renderFallbackValue: null,
		getCoreRowModel: getCoreRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
	})
	const filteredPage = filteredTable.getRowModel().rows
	expect(filteredTable.getPrePaginationRowModel().rows.length).toBeGreaterThan(1_000)
	expect(filteredPage).toHaveLength(50)
	expect(filteredPage.every((row) => row.original.name.includes("system-9"))).toBe(true)
})

test("pagination controls wrap to fit narrow mobile cards", () => {
	expect(paginationLayoutClassName.split(" ")).toContain("flex-wrap")
})

test("virtualized tables return to the first visible row when page rows change", () => {
	const scrollElement = { scrollTop: 1_200 }
	resetVirtualizedScrollPosition(scrollElement)
	expect(scrollElement.scrollTop).toBe(0)
})
