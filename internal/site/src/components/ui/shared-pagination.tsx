import { useStore } from "@nanostores/react"
import { ChevronLeftIcon, ChevronRightIcon, ChevronsLeftIcon, ChevronsRightIcon } from "lucide-react"
import { useCallback, useEffect, useId, useMemo, useState } from "react"
import { Trans } from "@lingui/react/macro"
import type { PaginationState, Table, Updater } from "@tanstack/react-table"
import { queueUserSettings } from "@/lib/api"
import { $authenticated, $userSettings } from "@/lib/stores"
import { Button } from "./button"
import { Label } from "./label"
import { defaultPageSize, normalizePageSize, pageSizes, paginationLayoutClassName, savePageSize } from "@/lib/pagination"

export { pageSizes }

function readSavedPageSize() {
	try {
		return normalizePageSize(localStorage.getItem("besz-rows-per-page"))
	} catch {
		return defaultPageSize
	}
}

export function usePagination() {
	const settings = useStore($userSettings, { keys: ["rowsPerPage"] })
	const pageSize = normalizePageSize(settings.rowsPerPage ?? readSavedPageSize())
	const [pageIndex, setPageIndex] = useState(0)
	const pagination = useMemo<PaginationState>(() => ({ pageIndex, pageSize }), [pageIndex, pageSize])
	const onPaginationChange = useCallback(
		(updater: Updater<PaginationState>) => {
			const next = typeof updater === "function" ? updater(pagination) : updater
			setPageIndex(next.pageSize === pageSize ? next.pageIndex : 0)
			if (next.pageSize !== pageSize) {
				savePageSize(localStorage, next.pageSize, (rowsPerPage) => {
					$userSettings.setKey("rowsPerPage", rowsPerPage)
				}, $authenticated.get() ? (rowsPerPage) => queueUserSettings({ rowsPerPage }) : undefined)
			}
		},
		[pagination, pageSize]
	)
	const resetPageIndex = useCallback(() => setPageIndex(0), [])
	return { pagination, onPaginationChange, resetPageIndex }
}

export function DataTablePagination<T>({ table }: { table: Table<T> }) {
	const selectId = useId()
	const { pageIndex, pageSize } = table.getState().pagination
	const pageCount = table.getPageCount()
	const total = table.getFilteredRowModel().rows.length
	useEffect(() => {
		const count = table.getPageCount()
		if (pageIndex >= count && pageIndex > 0) table.setPageIndex(Math.max(0, count - 1))
	}, [pageIndex, table, total])
	if (total <= Math.min(...pageSizes)) return null
	return (
		<div className={paginationLayoutClassName} aria-label="Table pagination">
			<div className="text-muted-foreground flex-1">
				<Trans>{table.getFilteredRowModel().rows.length} row(s)</Trans>
			</div>
			<div className="flex items-center gap-2">
				<Label className="sr-only" htmlFor={selectId}><Trans>Rows per page</Trans></Label>
				<select
					id={selectId}
					className="h-9 rounded-md border bg-background px-2"
					value={pageSize}
					onChange={(event) => table.setPageSize(Number(event.target.value))}
				>
					{pageSizes.map((size) => <option key={size} value={size}>{size}</option>)}
				</select>
			</div>
			<div className="whitespace-nowrap"><Trans>Page {pageIndex + 1} of {Math.max(1, pageCount)}</Trans></div>
			<div className="flex gap-1">
				<Button variant="outline" size="icon" className="size-9" aria-label="First page" onClick={() => table.setPageIndex(0)} disabled={!table.getCanPreviousPage()}><ChevronsLeftIcon className="size-4" /></Button>
				<Button variant="outline" size="icon" className="size-9" aria-label="Previous page" onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()}><ChevronLeftIcon className="size-4" /></Button>
				<Button variant="outline" size="icon" className="size-9" aria-label="Next page" onClick={() => table.nextPage()} disabled={!table.getCanNextPage()}><ChevronRightIcon className="size-4" /></Button>
				<Button variant="outline" size="icon" className="size-9" aria-label="Last page" onClick={() => table.setPageIndex(pageCount - 1)} disabled={!table.getCanNextPage()}><ChevronsRightIcon className="size-4" /></Button>
			</div>
		</div>
	)
}
