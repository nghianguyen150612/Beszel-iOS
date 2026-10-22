/** Returns finite, non-negative integer bytes safe for chart calculations. */
export function safeMemoryBytes(value?: number | null): number | undefined {
	if (value == null || !Number.isFinite(value) || value < 0) return undefined
	return Math.trunc(value)
}

/** Converts deprecated GB telemetry only when the exact byte field is absent. */
export function legacyMemoryBytes(gigabytes?: number | null): number | undefined {
	if (gigabytes == null || !Number.isFinite(gigabytes) || gigabytes < 0) return undefined
	return safeMemoryBytes(gigabytes * 1024 ** 3)
}

/** Exact bytes win over legacy GB values, including an explicitly measured zero. */
export function memoryFieldBytes(exact?: number | null, legacyGigabytes?: number | null) {
	return exact == null ? legacyMemoryBytes(legacyGigabytes) : safeMemoryBytes(exact)
}

/** True only when both swap counters are present and total capacity is positive. */
export function hasMeasuredSwap(totalBytes?: number | null, usedBytes?: number | null): boolean {
	const total = safeMemoryBytes(totalBytes)
	const used = safeMemoryBytes(usedBytes)
	return total !== undefined && total > 0 && used !== undefined
}
