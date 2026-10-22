import { expect, test } from "bun:test"
import { hasMeasuredSwap, legacyMemoryBytes, memoryFieldBytes, safeMemoryBytes } from "./memory"

test("keeps exact byte values for small and large hosts", () => {
	expect(safeMemoryBytes(64 * 1024 ** 2)).toBe(64 * 1024 ** 2)
	expect(safeMemoryBytes(1.5 * 1024 ** 4)).toBe(1.5 * 1024 ** 4)
	expect(safeMemoryBytes(2 ** 64)).toBe(2 ** 64)
})

test("converts only finite non-negative legacy values", () => {
	expect(legacyMemoryBytes(0.125)).toBe(128 * 1024 ** 2)
	expect(legacyMemoryBytes(0)).toBe(0)
	expect(legacyMemoryBytes(-1)).toBeUndefined()
	expect(legacyMemoryBytes(Number.NaN)).toBeUndefined()
	expect(legacyMemoryBytes(Number.POSITIVE_INFINITY)).toBeUndefined()
})

test("uses exact zero and rejects invalid exact values without legacy fallback", () => {
	expect(memoryFieldBytes(0, 5)).toBe(0)
	expect(memoryFieldBytes(Number.NaN, 5)).toBeUndefined()
	expect(memoryFieldBytes(undefined, 5)).toBe(5 * 1024 ** 3)
})

test("distinguishes measured zero swap use from unavailable swap", () => {
	expect(hasMeasuredSwap(4 * 1024 ** 3, 0)).toBe(true)
	expect(hasMeasuredSwap(0, 0)).toBe(false)
	expect(hasMeasuredSwap(undefined, 0)).toBe(false)
	expect(hasMeasuredSwap(4 * 1024 ** 3, undefined)).toBe(false)
	expect(hasMeasuredSwap(4 * 1024 ** 3, Number.NaN)).toBe(false)
})
