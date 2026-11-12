export const pageSizes = [10, 20, 50, 100, 200]
export const defaultPageSize = 20
export const paginationLayoutClassName = "mt-3 flex flex-wrap items-center gap-3 text-sm tabular-nums"

export function normalizePageSize(value: unknown): number {
	const size = Number(value)
	return pageSizes.includes(size) ? size : defaultPageSize
}

export function getPageBounds(totalRows: number, pageIndex: number, pageSize: number) {
	const count = Math.max(0, Math.ceil(totalRows / pageSize))
	const index = Math.min(Math.max(0, pageIndex), Math.max(0, count - 1))
	const start = Math.min(totalRows, index * pageSize)
	return { pageIndex: index, pageCount: count, start, end: Math.min(totalRows, start + pageSize) }
}

export function writeSavedPageSize(storage: Pick<Storage, "setItem">, value: unknown) {
	const normalized = normalizePageSize(value)
	storage.setItem("besz-rows-per-page", String(normalized))
	return normalized
}

export function savePageSize(
	storage: Pick<Storage, "setItem">,
	value: unknown,
	setUserSetting: (pageSize: number) => void,
	persistUserSetting?: (pageSize: number) => void
) {
	const normalized = normalizePageSize(value)
	try {
		writeSavedPageSize(storage, normalized)
	} catch {
		// Continue with the in-memory preference if browser storage is unavailable.
	}
	setUserSetting(normalized)
	persistUserSetting?.(normalized)
	return normalized
}

export function resetVirtualizedScrollPosition(element: Pick<HTMLElement, "scrollTop">) {
	element.scrollTop = 0
}
